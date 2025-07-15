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
}

type Config struct {
	ListenPort   string        `toml:"listen-port"`
	LogLevel     string        `toml:"log-level"`
	LogFile      string        `toml:"log-file"`
	ServerConfig *ServerConfig `toml:"server-config"`
	DBConfig     *DBConfig     `toml:"db-config"`
}

func NewConfig() *Config {
	return &Config{
		ListenPort:   "8000",
		LogLevel:     "info",
		LogFile:      "stdout",
		DBConfig:     &DBConfig{},
		ServerConfig: NewServerConfig(),
	}
}
