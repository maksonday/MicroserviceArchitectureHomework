package main

import (
	"delivery/config"
	"delivery/db"
	"delivery/logging"
	"delivery/metrics"
	"delivery/redis"
	"delivery/service"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//	@title			Delivery API
//	@version		1.0
//	@description	This is a delivery service API.
//	@termsOfService	http://swagger.io/terms/

//	@license.name	Apache 2.0

func main() {
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Printf("error getting executable path: %v\n", err)
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

	service.NewCourReserveProcessor(config)

	go service.GetCourReserveProcessor().Run()

	service.NewNotificationsProcessor(config)

	go service.GetNotificationsProcessor().Run()

	metrics.Init()
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8542", nil)

	server := service.NewServer(config)

	log.Fatalf("serve: %s", server.ListenAndServe(":"+config.ListenPort))
}
