package main

import (
	"fmt"
	"log"
	"order/config"
	"order/db"
	"order/logging"
	"order/redis"
	"order/service"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func main() {
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		return
	}

	appName := filepath.Base(executablePath)
	config := config.NewConfig()
	if _, err := toml.DecodeFile("/usr/local/etc/"+appName+".conf", config); err != nil {
		log.Fatalf("loading config: %s", err)
	}

	logging.Init(appName, config)

	if err := db.Init(config.DBConfig); err != nil {
		log.Fatalf("init database: %s", err)
	}

	redis.Init(config.RedisConfig)

	go func() {
		service.NewPaymentsProcessor(config)
		service.GetPaymentsProcessor().Run()
	}()

	go func() {
		service.NewStockProcessor(config)
		service.GetStockProcessor().Run()
	}()

	server := service.NewServer(config)

	log.Fatalf("serve: %s", server.ListenAndServe(":"+config.ListenPort))
}
