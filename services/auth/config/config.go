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

type Config struct {
	BillingAddr  string        `toml:"billing-addr"`
	BasePath     string        `toml:"base-path"`
	ListenPort   string        `toml:"listen-port"`
	LogLevel     string        `toml:"log-level"`
	LogFile      string        `toml:"log-file"`
	ServerConfig *ServerConfig `toml:"server-config"`
	DBMaster     *DBConfig     `toml:"db-master"`
	DBReplica    *DBConfig     `toml:"db-replica"`
	RedisMaster  *RedisConfig  `toml:"redis-master"`
	RedisReplica *RedisConfig  `toml:"redis-replica"`
}

func NewConfig() *Config {
	return &Config{
		BillingAddr: "arch.homework/billing",
		BasePath:    "auth",
		ListenPort:  "8000",
		LogLevel:    "info",
		LogFile:     "stdout",
		DBMaster: &DBConfig{
			Port: 5432,
		},
		DBReplica: &DBConfig{
			Port: 5432,
		},
		RedisMaster: &RedisConfig{
			Host: "redis",
			Port: 6379,
			DB:   0,
		},
		RedisReplica: &RedisConfig{
			Host: "redis",
			Port: 6379,
			DB:   0,
		},
		ServerConfig: NewServerConfig(),
	}
}
