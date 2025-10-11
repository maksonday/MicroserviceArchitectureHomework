package config

import (
	"time"
)

type ServerConfig struct {
	ReadTimeout  time.Duration `toml:"read-timeout"`
	WriteTimeout time.Duration `toml:"write-timeout"`
	IdleTimeout  time.Duration `toml:"idle-timeout"`
	Concurrency  int           `toml:"concurrency"`
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		Concurrency:  100,
	}
}

type DBConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	SSLMode  string `toml:"sslmode"`
}

type RetryConfig struct {
	Count uint64        `toml:"count"`
	Delay time.Duration `toml:"delay"`
}

func NewRetryConfig() *RetryConfig {
	return &RetryConfig{
		Count: 3,
		Delay: time.Second,
	}
}

type RedisConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
	DB   int    `toml:"db"`
}

type KafkaConsumerConfig struct {
	Brokers []string `toml:"brokers"`
	GroupID string   `toml:"group-id"`
	Topic   string   `toml:"topic"`
	Version string   `toml:"version"`
}

func NewKafkaConsumerConfig() *KafkaConsumerConfig {
	return &KafkaConsumerConfig{
		Brokers: []string{"kafka:9092"},
		GroupID: "payments",
		Topic:   "payments",
	}
}

type KafkaProducerConfig struct {
	Brokers []string `toml:"brokers"`
	Topic   string   `toml:"topic"`
	Version string   `toml:"version"`
}

func NewKafkaProducerConfig() *KafkaProducerConfig {
	return &KafkaProducerConfig{
		Brokers: []string{"kafka:9092"},
		Topic:   "payments_status",
	}
}

type Config struct {
	BasePath                    string               `toml:"base-path"`
	AuthAddr                    string               `toml:"auth-addr"`
	ListenPort                  string               `toml:"listen-port"`
	LogLevel                    string               `toml:"log-level"`
	LogFile                     string               `toml:"log-file"`
	ServerConfig                *ServerConfig        `toml:"server-config"`
	DBMaster                    *DBConfig            `toml:"db-master"`
	DBReplica                   *DBConfig            `toml:"db-replica"`
	DBRetry                     *RetryConfig         `toml:"db-retry"`
	RedisReplica                *RedisConfig         `toml:"redis-replica"`
	CourReserveConsumerConfig   *KafkaConsumerConfig `toml:"cour-reserve-consumer-config"`
	CourReserveProducerConfig   *KafkaProducerConfig `toml:"cour-reserve-producer-config"`
	NotificationsProducerConfig *KafkaProducerConfig `toml:"notifications-producer-config"`
}

func NewConfig() *Config {
	return &Config{
		BasePath:   "users",
		AuthAddr:   "arch.homework",
		ListenPort: "8000",
		LogLevel:   "info",
		LogFile:    "stdout",
		DBMaster: &DBConfig{
			Port: 5432,
		},
		DBReplica: &DBConfig{
			Port: 5432,
		},
		RedisReplica: &RedisConfig{
			Host: "redis",
			Port: 6379,
			DB:   0,
		},
		DBRetry:                     NewRetryConfig(),
		ServerConfig:                NewServerConfig(),
		CourReserveConsumerConfig:   NewKafkaConsumerConfig(),
		CourReserveProducerConfig:   NewKafkaProducerConfig(),
		NotificationsProducerConfig: NewKafkaProducerConfig(),
	}
}
