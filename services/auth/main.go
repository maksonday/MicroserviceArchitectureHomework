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

//	@title			Users API
//	@version		1.0
//	@description	This is auth service API.
//	@termsOfService	http://swagger.io/terms/

// 	@license.name	Apache 2.0

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

	server := service.NewServer(config)

	log.Fatalf("serve: %s", server.ListenAndServe(":"+config.ListenPort))
}
