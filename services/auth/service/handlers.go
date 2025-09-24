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

var (
	ErrCreateUser         = errors.New("create user error")
	ErrCreateAccount      = errors.New("create account error")
	ErrBadInput           = errors.New("bad input")
	ErrIssueTokens        = errors.New("issue tokens error")
	ErrGetUserCredentials = errors.New("can't get credentials")
	ErrIncorrectPasword   = errors.New("incorrect password")
	ErrNoAccessToken      = errors.New("no access token")
	ErrNoRefreshToken     = errors.New("no refresh token")
	ErrBadAccessToken     = errors.New("bad access token")
	ErrBadRefreshToken    = errors.New("bad refresh token")
	ErrRefreshToken       = errors.New("refresh token error")
)

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
func registerHandler(ctx *fasthttp.RequestCtx, billingAddr string) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var user types.User

	if err := json.Unmarshal(ctx.Request.Body(), &user); err != nil {
		zap.L().Error(fmt.Errorf("unmarshal user: %w", err).Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusUnauthorized)
		return
	}

	id, err := db.CreateUser(&user)
	if err != nil {
		zap.L().Error(fmt.Errorf("create user: %w", err).Error())
		handleError(ctx, ErrCreateUser, fasthttp.StatusUnauthorized)
		return
	}

	user.Id = id
	if err := issueTokens(ctx, &user); err != nil {
		zap.L().Error(fmt.Errorf("issue tokens: %w", err).Error())
		handleError(ctx, ErrIssueTokens, fasthttp.StatusUnauthorized)
		return
	}

	if err := createAccount(ctx, billingAddr, id); err != nil {
		db.DeleteUser(id)
		zap.L().Error(fmt.Errorf("create account: %w", err).Error())
		handleError(ctx, ErrCreateAccount, fasthttp.StatusUnauthorized)
		return
	}
}

func createAccount(ctx *fasthttp.RequestCtx, billingAddr string, userId int64) error {
	req := fasthttp.AcquireRequest()
	// берем из подготовленного response заголовки и куки
	authHeader := string(ctx.Response.Header.Peek("Authorization"))
	refreshToken := string(ctx.Response.Header.PeekCookie(refreshCookieName))
	// Set header authorization
	req.Header.Set("Authorization", authHeader)
	// Set cookie refresh_token
	req.Header.SetCookie(refreshCookieName, refreshToken)
	req.SetRequestURI(fmt.Sprintf("%s/create_account", billingAddr))
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	zap.L().Info("creating account", zap.Int64("user_id", userId))

	if err := fasthttp.Do(req, resp); err != nil {
		return fmt.Errorf("failed to send create account request: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusCreated {
		return errors.New("failed to create account: " + string(resp.Body()))
	}

	return nil
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
		zap.L().Error(fmt.Errorf("unmarshal input: %w", err).Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusUnauthorized)
		return
	}

	user, err := db.GetUserCredentials(input.Username)
	if err != nil {
		zap.L().Error(fmt.Errorf("get user credentials: %w", err).Error())
		handleError(ctx, ErrGetUserCredentials, fasthttp.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		zap.L().Error(ErrIncorrectPasword.Error())
		handleError(ctx, ErrIncorrectPasword, fasthttp.StatusUnauthorized)
		return
	}

	if err := issueTokens(ctx, user); err != nil {
		zap.L().Error(fmt.Errorf("issue tokens: %w", err).Error())
		handleError(ctx, ErrIssueTokens, fasthttp.StatusUnauthorized)
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
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/logout [get]
func logoutHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		zap.L().Error(ErrNoAccessToken.Error())
		handleError(ctx, ErrNoAccessToken, fasthttp.StatusUnauthorized)
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	claimsAt, err := parseToken(accessToken)
	if err != nil {
		zap.L().Error(fmt.Errorf("failed to parse access-token: %w", err).Error())
		handleError(ctx, ErrBadAccessToken, fasthttp.StatusUnauthorized)
		return
	}

	if err := redis.Client.PutTokenToBlacklist(blAtKeyPrefix, claimsAt); err != nil {
		zap.L().Error(fmt.Errorf("failed to blacklist access-token: %w", err).Error())
		handleError(ctx, ErrBadAccessToken, fasthttp.StatusUnauthorized)
		return
	}

	refreshToken := retrieveRefreshTokenFromCtx(ctx)
	if len(refreshToken) == 0 {
		zap.L().Error(ErrNoRefreshToken.Error())
		handleError(ctx, ErrNoRefreshToken, fasthttp.StatusUnauthorized)
		return
	}

	claimsRt, err := parseToken(refreshToken)
	if err != nil {
		zap.L().Error(fmt.Errorf("failed to parse refresh-token: %w", err).Error())
		handleError(ctx, ErrBadRefreshToken, fasthttp.StatusUnauthorized)
		return
	}

	if err := redis.Client.PutTokenToBlacklist(blRtKeyPrefix, claimsRt); err != nil {
		zap.L().Error(fmt.Errorf("failed to blacklist access-token: %w", err).Error())
		handleError(ctx, ErrBadAccessToken, fasthttp.StatusUnauthorized)
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
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/refresh [get]
func refreshHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	refreshToken := retrieveRefreshTokenFromCtx(ctx)
	if len(refreshToken) == 0 {
		zap.L().Error(ErrNoRefreshToken.Error())
		handleError(ctx, ErrNoRefreshToken, fasthttp.StatusUnauthorized)
		return
	}

	claims, err := parseToken(refreshToken)
	if err != nil {
		zap.L().Error(fmt.Errorf("failed to parse refresh token: %w", err).Error())
		handleError(ctx, ErrBadRefreshToken, fasthttp.StatusUnauthorized)
		return
	}

	// проверяем, нет ли refresh-token в блэклисте(из-за logout, например)
	blacklisted, err := redis.Client.CheckTokenBlacklist(blRtKeyPrefix, claims)
	if err != nil && !errors.Is(err, redis.ErrNil) {
		zap.L().Error(fmt.Errorf("failed to check refresh token blacklist: %w", err).Error())
		handleError(ctx, ErrBadRefreshToken, fasthttp.StatusUnauthorized)
		return
	}

	if blacklisted {
		zap.L().Error("refresh token is blacklisted")
		handleError(ctx, ErrBadAccessToken, fasthttp.StatusUnauthorized)
		return
	}

	username, ok := claims["username"].(string)
	if !ok {
		zap.L().Error("failed to get data from refresh token")
		handleError(ctx, ErrBadRefreshToken, fasthttp.StatusUnauthorized)
		return
	}

	// достаем данные пользователя для генерации access-token
	user, err := db.GetUserCredentials(username)
	if err != nil {
		zap.L().Error(fmt.Errorf("failed to get user: %w", err).Error())
		handleError(ctx, ErrGetUserCredentials, fasthttp.StatusUnauthorized)
		return
	}

	accessToken, err := generateAccessToken(user)
	if err != nil {
		zap.L().Error(fmt.Errorf("failed to generate access token: %w", err).Error())
		handleError(ctx, ErrRefreshToken, fasthttp.StatusUnauthorized)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(types.RefreshResponse{AccessToken: accessToken})
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
