package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"users/db"
	"users/redis"
	"users/types"

	"github.com/golang-jwt/jwt"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const (
	blAtKeyPrefix = "bl:at:"
	blRtKeyPrefix = "bl:rt:"
)

func init() {
	data, err := os.ReadFile("/keys/public.pem")
	if err != nil {
		log.Fatalf("failed to read public.pem: %v", err)
	}
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		log.Fatalf("invalid public key: %v", err)
	}
}

func healthCheckHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.WriteString(`{"status":"OK"}`)
}

func handleUser(ctx *fasthttp.RequestCtx, userIdStr string) {
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		handleError(ctx, err)
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		getUser(ctx, userId)
	case fasthttp.MethodPost:
		updateUser(ctx, userId)
	case fasthttp.MethodDelete:
		deleteUser(ctx, userId)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func createUser(ctx *fasthttp.RequestCtx) {
	var user types.User
	if err := json.Unmarshal(ctx.Request.Body(), &user); err != nil {
		handleError(ctx, err)
		return
	}

	id, err := db.CreateUser(&user)
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetContentType("application/json")
	response, err := json.Marshal(map[string]any{"id": id})
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.Write(response)
}

func deleteUser(ctx *fasthttp.RequestCtx, userId int64) {
	if err := db.DeleteUser(userId); err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

func getUser(ctx *fasthttp.RequestCtx, userId int64) {
	user, err := db.GetUser(userId)
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetContentType("application/json")
	response, err := json.Marshal(user)
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(response)
}

func updateUser(ctx *fasthttp.RequestCtx, userId int64) {
	var user types.User
	if err := json.Unmarshal(ctx.Request.Body(), &user); err != nil {
		handleError(ctx, err)
		return
	}

	user.Id = userId
	if err := db.UpdateUser(&user); err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func handleError(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(fasthttp.StatusBadRequest)
	ctx.SetContentType("application/json")
	response := map[string]string{"error": err.Error()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		zap.L().Error("failed to marshal error response: " + err.Error())
		return
	}
	ctx.Write(jsonResponse)
}

func AuthMiddleware() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		authHeader := string(ctx.Request.Header.Peek("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.Redirect("/login", fasthttp.StatusFound)
			return
		}

		accessToken := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := parseToken(accessToken)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				ctx.Redirect("/auth/refresh", fasthttp.StatusFound)
				return
			}

			handleError(ctx, fmt.Errorf("failed to parse access-token: %w", err))
			return
		}

		// Проверка в Redis
		exists, err := redis.Client.CheckTokenBlacklist(blAtKeyPrefix, claims)
		if err != nil {
			ctx.Error("redis error", fasthttp.StatusInternalServerError)
			return
		}

		// Был сделан logout, нужно логиниться заново
		if exists {
			ctx.Redirect("/auth/login", fasthttp.StatusFound)
			return
		}

	}
}
