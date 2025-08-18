package main

import (
	"auth/config"
	"auth/db"
	"auth/logging"
	"auth/redis"
	"auth/service"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func main() {
	appName := filepath.Base(os.Args[0])
	config := config.NewConfig()
	if _, err := toml.DecodeFile("/usr/local/etc/"+appName+".conf", config); err != nil {
		log.Fatalf("loading config: %s", err)
	}

	logging.Init(appName, config)

	if err := db.Init(config.DBConfig); err != nil {
		log.Fatalf("init database: %s", err)
	}

	redis.Init(config.RedisConfig)

	server := service.NewServer(config.ServerConfig)

	log.Fatalf("serve: %s", server.ListenAndServe(":"+config.ListenPort))
}
