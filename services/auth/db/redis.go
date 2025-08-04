package db

import (
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func init() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // для k8s — имя Redis-сервиса
		Password: "",           // если есть
		DB:       0,
	})
}
