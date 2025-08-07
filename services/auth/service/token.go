package service

import (
	"crypto/rsa"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

var privateKey *rsa.PrivateKey

func init() {
	data, err := os.ReadFile("/keys/private.pem")
	if err != nil {
		log.Fatalf("failed to read private.pem: %v", err)
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
		"exp":      time.Now().Add(accessTokenExp),
		"username": username,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func generateRefreshToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"jti":      uuid.NewString(),
		"exp":      time.Now().Add(refreshTokenExp),
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

	ctx.SetContentType("application/json")
	return json.NewEncoder(ctx).Encode(map[string]string{
		"access_token": accessToken,
	})
}

func parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// определяется из сертификата
		return privateKey, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	// достаем claim из refresh-token
	return token.Claims.(jwt.MapClaims), nil
}
