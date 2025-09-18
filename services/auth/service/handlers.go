package service

import (
	"auth/db"
	"auth/redis"
	"auth/types"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenExp  = 15 * time.Minute
	refreshTokenExp = 30 * 24 * time.Hour // 1 месяц
	blAtKeyPrefix   = "bl:at:"
	blRtKeyPrefix   = "bl:rt:"
)

func healthCheckHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.WriteString(`{"status":"OK"}`)
}

// register godoc
//
//	@Summary		register user
//	@Description	register user
//	@Tags			auth
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/register [post]
func registerHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var user types.User

	if err := json.Unmarshal(ctx.Request.Body(), &user); err != nil {
		handleError(ctx, fmt.Errorf("unmarshal user: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	id, err := db.CreateUser(&user)
	if err != nil {
		handleError(ctx, fmt.Errorf("create user: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	if err := issueTokens(ctx, id, user.Username); err != nil {
		handleError(ctx, fmt.Errorf("issue tokens: %w", err), fasthttp.StatusUnauthorized)
		return
	}
}

// register godoc
//
//	@Summary		register user
//	@Description	register user
//	@Tags			auth
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		401	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/login [post]
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
		handleError(ctx, fmt.Errorf("unmarshal input: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	user, err := db.GetUserCredentials(input.Username)
	if err != nil {
		handleError(ctx, fmt.Errorf("get user credentials: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		handleError(ctx, fmt.Errorf("password incorrect"), fasthttp.StatusUnauthorized)
		return
	}

	if err := issueTokens(ctx, user.Id, user.Username); err != nil {
		handleError(ctx, fmt.Errorf("issue tokens: %w", err), fasthttp.StatusUnauthorized)
	}
}

// logout godoc
//
//	@Summary		logout user
//	@Description	logout user
//	@Tags			auth
//	@Produce 		json
//	@Success		200	{object}	types.LogoutResponse
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/logout [get]
func logoutHandler(ctx *fasthttp.RequestCtx) {
	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		handleError(ctx, fmt.Errorf("no access-token"), fasthttp.StatusUnauthorized)
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	claimsAt, err := parseToken(accessToken)
	if err != nil {
		handleError(ctx, fmt.Errorf("failed to parse access-token: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	if err := redis.Client.PutTokenToBlacklist(blAtKeyPrefix, claimsAt); err != nil {
		handleError(ctx, fmt.Errorf("failed to blacklist access-token: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	refreshToken := retrieveRefreshTokenFromCtx(ctx)
	if len(refreshToken) == 0 {
		handleError(ctx, fmt.Errorf("no refresh token"), fasthttp.StatusUnauthorized)
		return
	}

	claimsRt, err := parseToken(refreshToken)
	if err != nil {
		handleError(ctx, fmt.Errorf("failed to parse refresh-token: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	if err := redis.Client.PutTokenToBlacklist(blRtKeyPrefix, claimsRt); err != nil {
		handleError(ctx, fmt.Errorf("failed to blacklist access-token: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	ctx.Response.Header.SetCookie(getEmptyCookie())
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte(`{"message":"logged out"}`))
}

// refresh godoc
//
//	@Summary		refresh user tokens
//	@Description	refresh user tokens
//	@Tags			auth
//	@Produce 		json
//	@Success		200	{object}	types.RefreshResponse
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/refresh [get]
func refreshHandler(ctx *fasthttp.RequestCtx) {
	refreshToken := retrieveRefreshTokenFromCtx(ctx)
	if len(refreshToken) == 0 {
		handleError(ctx, fmt.Errorf("no refresh token"), fasthttp.StatusUnauthorized)
		return
	}

	claims, err := parseToken(refreshToken)
	if err != nil {
		handleError(ctx, fmt.Errorf("failed to parse token: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	// проверяем, нет ли refresh-token в блэклисте(из-за logout, например)
	blacklisted, err := redis.Client.CheckTokenBlacklist(blRtKeyPrefix, claims)
	if err != nil && !errors.Is(err, redis.ErrNil) {
		handleError(ctx, fmt.Errorf("failed to check refresh token blacklist: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	if blacklisted {
		handleError(ctx, fmt.Errorf("refresh token is blacklisted"), fasthttp.StatusUnauthorized)
		return
	}

	username, ok := claims["username"].(string)
	if !ok {
		handleError(ctx, fmt.Errorf("failed to get data from refresh token"), fasthttp.StatusUnauthorized)
		return
	}

	// достаем данные пользователя для генерации access-token
	user, err := db.GetUserCredentials(username)
	if err != nil {
		handleError(ctx, fmt.Errorf("failed to get user: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	accessToken, err := generateAccessToken(user.Id, username)
	if err != nil {
		handleError(ctx, fmt.Errorf("failed to generate access token: %w", err), fasthttp.StatusUnauthorized)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(types.RefreshResponse{AccessToken: accessToken})
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	zap.L().Error(err.Error())
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
