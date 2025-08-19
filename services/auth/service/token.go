package service

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

var privateKey *rsa.PrivateKey

var ErrTokenInvalid = errors.New("invalid token")
var ErrTokenExpired = errors.New("access token expired")

func init() {
	data, err := os.ReadFile("/keys/cert.pem")
	if err != nil {
		log.Fatalf("failed to read private key: %v", err)
	}
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}
}

func generateAccessToken(userID int64, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"jti":      uuid.NewString(),
		"exp":      float64(time.Now().Add(accessTokenExp).Unix()),
		"username": username,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func generateRefreshToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"jti":      uuid.NewString(),
		"exp":      float64(time.Now().Add(refreshTokenExp).Unix()),
		"username": username,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func issueTokens(ctx *fasthttp.RequestCtx, userID int64, username string) error {
	// Access token
	accessToken, err := generateAccessToken(userID, username)
	if err != nil {
		return err
	}

	// Refresh token
	refreshToken, err := generateRefreshToken(username)
	if err != nil {
		return err
	}

	// Set-Cookie
	ctx.Response.Header.SetCookie(getCookieWithRefreshToken(refreshToken))
	// Set access token in header
	ctx.Response.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	return nil
}

func parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// определяется из сертификата
		return privateKey.Public(), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	// достаем claim из refresh-token
	return token.Claims.(jwt.MapClaims), nil
}
