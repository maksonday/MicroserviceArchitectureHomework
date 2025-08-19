package service

import (
	"crypto/rsa"
	"errors"
	"log"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

var publicKey *rsa.PublicKey

var ErrTokenExpired = errors.New("access token expired")
var ErrTokenInvalid = errors.New("invalid token")

func init() {
	data, err := os.ReadFile("/keys/cert.pem.pub")
	if err != nil {
		log.Fatalf("failed to read public key: %v", err)
	}
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		log.Fatalf("invalid public key: %v", err)
	}
}

func parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// определяется из сертификата
		return publicKey, nil
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
