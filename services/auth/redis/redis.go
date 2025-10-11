package redis

import (
	"auth/config"
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
	Master    *RedisClient
	Slave     *RedisClient
	redisOnce sync.Once

	ErrBadClaim = errors.New("bad claim error")
)

func Init(config *config.Config) {
	redisOnce.Do(func() {
		Master = &RedisClient{
			redis: redis.NewClient(&redis.Options{
				Addr:     config.RedisMaster.Host + ":" + strconv.Itoa(config.RedisMaster.Port),
				Password: getRedisPassword(),
				DB:       config.RedisMaster.DB,
			}),
		}
		Slave = &RedisClient{
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

func PutTokenToBlacklist(prefix string, claims jwt.MapClaims) error {
	jti, ok := claims["jti"].(string)
	if !ok {
		return ErrBadClaim
	}

	username, ok := claims["username"].(string)
	if !ok {
		return ErrBadClaim
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return ErrBadClaim
	}

	ttl := time.Until(time.Unix(int64(exp), 0))
	if ttl <= 0 {
		ttl = time.Minute * 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return Master.redis.Set(ctx, prefix+jti+":"+username, true, ttl).Err()
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

	return Slave.redis.Get(ctx, prefix+jti+":"+username).Bool()
}
