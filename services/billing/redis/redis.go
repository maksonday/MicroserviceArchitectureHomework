package redis

import (
	"billing/config"
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const timeout = 5 * time.Second

const ErrNil = redis.Nil

type RedisClient struct {
	redis *redis.Client
}

var (
	slave     *RedisClient
	redisOnce sync.Once

	ErrBadClaim = errors.New("bad claim error")
)

func Init(config *config.Config) {
	redisOnce.Do(func() {
		slave = &RedisClient{
			redis: redis.NewClient(&redis.Options{
				Addr:     config.RedisReplica.Host + ":" + strconv.Itoa(config.RedisReplica.Port),
				Password: getRedisPassword(),
				DB:       config.RedisReplica.DB,
			}),
		}
	})
}

func getRedisPassword() string {
	data, err := os.ReadFile("/secret/redis/password")
	if err != nil {
		log.Fatalf("failed to read password file: %s", err)
	}

	return string(data)
}

func CheckTokenBlacklist(prefix string, claims jwt.MapClaims) (bool, error) {
	jti, ok := claims["jti"].(string)
	if !ok {
		return false, ErrBadClaim
	}

	username, ok := claims["username"].(string)
	if !ok {
		return false, ErrBadClaim
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return slave.redis.Get(ctx, prefix+jti+":"+username).Bool()
}
