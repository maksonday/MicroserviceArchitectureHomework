package main

import (
	"fmt"
	"log"
	"net/http"
	"notifications/config"
	"notifications/db"
	"notifications/logging"
	"notifications/metrics"
	"notifications/redis"
	"notifications/service"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//	@title			Notifications API
//	@version		1.0
//	@description	This is notifications service API.
//	@termsOfService	http://swagger.io/terms/

//	@license.name	Apache 2.0

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

	if err := db.Init(config); err != nil {
		log.Fatalf("init database: %s", err)
	}

	redis.Init(config)

	go func() {
		service.NewNotificationsProcessor(config).Run()
	}()

	metrics.Init()
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8542", nil)

	server := service.NewServer(config)

	log.Fatalf("serve: %s", server.ListenAndServe(":"+config.ListenPort))
}
