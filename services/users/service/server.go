package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"users/config"
	"users/redis"

	"github.com/valyala/fasthttp"
)

const maxBodySize = 16 << 20 // 16 MB

var ErrNoAccessToken = errors.New("no access-token")
var ErrRedis = errors.New("redis error")

func NewServer(config *config.Config) *fasthttp.Server {
	basePath := strings.Trim(config.BasePath, "/")
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			parts := strings.Split(strings.Trim(string(ctx.Path()), "/"), "/")
			if len(parts) < 2 || parts[0] != basePath {
				ctx.Error("not found", fasthttp.StatusNotFound)
				return
			}

			switch parts[1] {
			case "user":
				switch {
				case len(parts) == 2:
					var (
						userId int64
						err    error
					)

					if userId, err = authMiddleware(config.AuthAddr, ctx); err != nil {
						handleError(ctx, err, fasthttp.StatusUnauthorized)
						return
					}

					handleUser(ctx, userId)
				default:
					ctx.Error("not found", fasthttp.StatusNotFound)
					return
				}
			case "health":
				healthCheckHandler(ctx)
			default:
				ctx.Error("not found", fasthttp.StatusNotFound)
			}
		},

		MaxRequestBodySize: maxBodySize,
		ReadTimeout:        config.ServerConfig.ReadTimeout,
		WriteTimeout:       config.ServerConfig.WriteTimeout,
		IdleTimeout:        config.ServerConfig.IdleTimeout,
		Concurrency:        config.ServerConfig.Concurrency,
	}

	return s
}

func authMiddleware(addr string, ctx *fasthttp.RequestCtx) (int64, error) {
	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, ErrNoAccessToken
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := parseToken(accessToken)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) {
			if err = refreshToken(addr, ctx); err != nil {
				return 0, fmt.Errorf("failed to refresh token: %w", err)
			}
		}

		return 0, fmt.Errorf("failed to parse access-token: %w", err)
	}

	// Проверка в Redis
	exists, err := redis.Client.CheckTokenBlacklist(blAtKeyPrefix, claims)
	if err != nil {
		return 0, ErrRedis
	}

	// Был сделан logout, нужно логиниться заново
	if exists {
		return 0, ErrNoAccessToken
	}

	if userId, ok := claims["user_id"]; !ok {
		return 0, fmt.Errorf("user_id not found in claims")
	} else {
		if _, ok := userId.(int64); !ok {
			return 0, fmt.Errorf("user_id is not an int64")
		}

		return userId.(int64), nil
	}
}

const refreshCookieName = "refresh_token"

func refreshToken(authAddr string, ctx *fasthttp.RequestCtx) error {
	req := fasthttp.AcquireRequest()
	req.Header.SetCookie(refreshCookieName, string(ctx.Request.Header.Cookie(refreshCookieName)))
	req.SetRequestURI(fmt.Sprintf("%s/refresh", authAddr))
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.Do(req, resp); err != nil {
		return fmt.Errorf("failed to send refresh token request: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("failed to refresh token")
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body(), &body); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if accessToken, ok := body["access_token"]; !ok {
		return fmt.Errorf("access_token not found in response body")
	} else {
		// Обновляем токен в заголовках
		ctx.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	return nil
}
