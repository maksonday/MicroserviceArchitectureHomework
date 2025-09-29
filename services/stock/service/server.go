package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"stock/config"
	"stock/redis"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const maxBodySize = 16 << 20 // 16 MB

var ErrNoAccessToken = errors.New("no access-token")
var ErrAccessTokenExpired = errors.New("access-token expired")
var ErrRedis = errors.New("redis error")
var ErrParseAccessToken = errors.New("failed to parse access-token")

func NewServer(config *config.Config) *fasthttp.Server {
	basePath := strings.Trim(config.BasePath, "/")
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			parts := strings.Split(strings.Trim(string(ctx.Path()), "/"), "/")
			if len(parts) < 2 || parts[0] != basePath {
				ctx.Error("not found", fasthttp.StatusNotFound)
				return
			}

			if len(parts) != 2 {
				ctx.Error("not found", fasthttp.StatusNotFound)
				return
			}

			switch parts[1] {
			case "get_items":
				handleGetItems(ctx)
			case "add_item", "update_item", "stock_change", "get_stock_changes":
				var (
					isAdmin bool
					err     error
				)

				if isAdmin, err = authMiddleware(config.AuthAddr, ctx); err != nil {
					handleError(ctx, err, fasthttp.StatusUnauthorized)
					return
				}

				if !isAdmin {
					handleError(ctx, errors.New("user is not admin"), fasthttp.StatusForbidden)
					return
				}

				switch parts[1] {
				case "add_item":
					handleAddItem(ctx)
				case "update_item":
					handleUpdateItem(ctx)
				case "stock_change":
					handleStockChange(ctx)
				case "get_stock_changes":
					handleGetStockChanges(ctx)
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

func authMiddleware(addr string, ctx *fasthttp.RequestCtx) (bool, error) {
	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false, ErrNoAccessToken
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := parseToken(accessToken)
	if err != nil {
		if !errors.Is(err, ErrTokenExpired) {
			return false, fmt.Errorf("failed to parse access token: %w", err)
		}

		if accessToken, err = refreshToken(addr, ctx); err != nil {
			return false, fmt.Errorf("failed to refresh token: %w", err)
		}

		if claims, err = parseToken(accessToken); err != nil {
			return false, ErrParseAccessToken
		}
	}

	// Проверка в Redis
	exists, err := redis.Client.CheckTokenBlacklist(blAtKeyPrefix, claims)
	if err != nil && !errors.Is(err, redis.ErrNil) {
		return false, ErrRedis
	}

	// Был сделан logout, нужно логиниться заново
	if exists {
		return false, ErrAccessTokenExpired
	}

	if userId, ok := claims["user_id"]; !ok {
		return false, fmt.Errorf("user_id not found in claims")
	} else {
		if _, ok := userId.(float64); !ok {
			return false, fmt.Errorf("user_id is not a float64")
		}

		return checkIsAdmin(claims), nil
	}
}

func checkIsAdmin(claims jwt.MapClaims) bool {
	roles, ok := claims["roles"]
	if !ok {
		zap.L().Warn("roles not found in claims")
		return false
	}

	rolesSlice, ok := roles.(string)
	if !ok {
		zap.L().Warn("roles is not a string")
		// Print actual type
		zap.L().Warn("actual type", zap.String("type", fmt.Sprintf("%T", roles)))
		return false
	}

	return slices.Contains(strings.Split(rolesSlice, ","), "admin")
}

const refreshCookieName = "refresh_token"

func refreshToken(authAddr string, ctx *fasthttp.RequestCtx) (string, error) {
	req := fasthttp.AcquireRequest()
	refreshToken := string(ctx.Request.Header.Cookie(refreshCookieName))
	req.Header.SetCookie(refreshCookieName, refreshToken)
	req.SetRequestURI(fmt.Sprintf("%s/refresh", authAddr))
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	zap.L().Info("refreshing access token", zap.String("refresh_token", refreshToken))

	if err := fasthttp.Do(req, resp); err != nil {
		return "", fmt.Errorf("failed to send refresh token request: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return "", fmt.Errorf("failed to refresh token")
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body(), &body); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	var (
		accessToken string
		ok          bool
	)

	if accessToken, ok = body["access_token"]; !ok {
		return "", fmt.Errorf("access_token not found in response body")
	}

	// Обновляем токен в заголовках
	ctx.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	ctx.Response.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	return accessToken, nil
}
