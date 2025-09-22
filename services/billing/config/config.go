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
	BasePath       string               `toml:"base-path"`
	AuthAddr       string               `toml:"auth-addr"`
	ListenPort     string               `toml:"listen-port"`
	LogLevel       string               `toml:"log-level"`
	LogFile        string               `toml:"log-file"`
	ServerConfig   *ServerConfig        `toml:"server-config"`
	DBConfig       *DBConfig            `toml:"db-config"`
	RedisConfig    *RedisConfig         `toml:"redis-config"`
	ConsumerConfig *KafkaConsumerConfig `toml:"consumer-config"`
	ProducerConfig *KafkaProducerConfig `toml:"producer-config"`
}

func NewConfig() *Config {
	return &Config{
		BasePath:   "users",
		AuthAddr:   "arch.homework",
		ListenPort: "8000",
		LogLevel:   "info",
		LogFile:    "stdout",
		DBConfig: &DBConfig{
			Port: 5432,
		},
		RedisConfig: &RedisConfig{
			Host: "redis",
			Port: 6379,
			DB:   0,
		},
		ServerConfig:   NewServerConfig(),
		ConsumerConfig: NewKafkaConsumerConfig(),
		ProducerConfig: NewKafkaProducerConfig(),
	}
}
