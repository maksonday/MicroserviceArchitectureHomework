package main

import (
	"log"
	"miniapp/config"
	"miniapp/db"
	"miniapp/logging"
	"miniapp/service"

	"github.com/BurntSushi/toml"
)

const appName = "miniapp"

func main() {
	config := config.NewConfig()
	if _, err := toml.DecodeFile("/usr/local/etc/"+appName+".conf", config); err != nil {
		log.Fatalf("loading config: %s", err)
	}

	logging.Init(appName, config)

	if err := db.Init(config.DBConfig); err != nil {
		log.Fatalf("init database: %s", err)
	}

	server := service.NewServer(config.ServerConfig)

	log.Fatalf("serve: %s", server.ListenAndServe(":"+config.ListenPort))
}
