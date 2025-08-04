package service

import (
	"context"
	"encoding/json"
	"log"
	"miniapp/db"
	"miniapp/types"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
)

const tokenExp = 15 * time.Minute

var jwtSecret []byte

func init() {
	data, err := os.ReadFile("/secret/jwt")
	if err != nil {
		log.Fatalf("failed to read jwt secret: %s", err)
	}

	jwtSecret = data
}

func getMapClaims(user *types.User) jwt.MapClaims {
	return jwt.MapClaims{
		"user_id": user.Id,
		"email":   user.Email,
		"exp":     time.Now().Add(tokenExp).Unix(),
		"jti":     uuid.NewString(),
	}
}

func registerHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var (
		user types.User
		err  error
	)

	if err := json.Unmarshal(ctx.Request.Body(), &user); err != nil {
		handleError(ctx, err)
		return
	}

	user.Id, err = db.CreateUser(&user)
	if err != nil {
		handleError(ctx, err)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, getMapClaims(&user))

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]string{"token": tokenString})
}

func loginHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(ctx.Request.Body(), &input); err != nil {
		handleError(ctx, err)
		return
	}

	user, err := db.GetUserCredentials(input.Username)
	if err != nil {
		ctx.Error("invalid credentials", fasthttp.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		ctx.Error("invalid credentials", fasthttp.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, getMapClaims(user))

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]string{"token": tokenString})
}

func logoutHandler(ctx *fasthttp.RequestCtx) {
	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		ctx.Error("unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		ctx.Error("unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	claims := token.Claims.(jwt.MapClaims)
	jti := claims["jti"].(string)
	exp := int64(claims["exp"].(float64))

	ttl := time.Until(time.Unix(exp, 0))
	if ttl <= 0 {
		ttl = time.Minute * 1
	}

	ctxRedis, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.RedisClient.Set(ctxRedis, "bl:"+jti, "true", ttl).Err()
	if err != nil {
		ctx.Error("redis error", fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte(`{"message":"logged out"}`))
}

func AuthMiddleware() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		authHeader := string(ctx.Request.Header.Peek("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.Redirect("/login", fasthttp.StatusFound)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			ctx.Redirect("/login", fasthttp.StatusFound)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			ctx.Redirect("/login", fasthttp.StatusFound)
			return
		}

		jti := claims["jti"].(string)

		// Проверка в Redis
		ctxRedis, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		exists, err := db.RedisClient.Exists(ctxRedis, "bl:"+jti).Result()
		if err != nil {
			ctx.Error("redis error", fasthttp.StatusInternalServerError)
			return
		}
		if exists > 0 {
			ctx.Redirect("/login", fasthttp.StatusFound)
			return
		}

		// Авторизация прошла
		userID := int(claims["user_id"].(int64))
		ctx.SetUserValue("userID", userID)
	}
}
