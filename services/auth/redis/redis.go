package redis

import (
	"auth/config"
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const timeout = 5 * time.Second

type RedisClient struct {
	redis *redis.Client
}

var (
	Client    *RedisClient
	redisOnce sync.Once

	ErrBadClaim = errors.New("bad claim error")
)

func Init(config *config.RedisConfig) {
	redisOnce.Do(func() {
		Client = &RedisClient{
			redis: redis.NewClient(&redis.Options{
				Addr:     config.Addr,
				Password: getRedisPassword(),
				DB:       config.DB,
			}),
		}
	})
}

func getRedisPassword() string {
	data, err := os.ReadFile("/secret/redis_password")
	if err != nil {
		log.Fatalf("failed to read password file: %s", err)
	}

	return string(data)
}

func (client *RedisClient) PutTokenToBlacklist(prefix string, claims jwt.MapClaims) error {
	jti, ok := claims["jti"].(string)
	if !ok {
		return ErrBadClaim
	}

	username, ok := claims["username"].(string)
	if !ok {
		return ErrBadClaim
	}

	exp, ok := claims["exp"].(int64)
	if !ok {
		return ErrBadClaim
	}

	ttl := time.Until(time.Unix(exp, 0))
	if ttl <= 0 {
		ttl = time.Minute * 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return Client.redis.Set(ctx, prefix+jti+":"+username, true, ttl).Err()
}

func (client *RedisClient) CheckTokenBlacklist(prefix string, claims jwt.MapClaims) (bool, error) {
	jti, ok := claims["jti"].(string)
	if !ok {
		return false, ErrBadClaim
	}

	username, ok := claims["username"].(string)
	if !ok {
		return false, ErrBadClaim
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return Client.redis.Get(ctx, prefix+jti+":"+username).Bool()
}
