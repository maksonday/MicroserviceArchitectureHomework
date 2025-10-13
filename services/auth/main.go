package main

import (
	"auth/config"
	"auth/db"
	"auth/logging"
	"auth/metrics"
	"auth/redis"
	"auth/service"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//	@title			Auth API
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

	if err := db.Init(config); err != nil {
		log.Fatalf("init database: %s", err)
	}

	redis.Init(config)

	metrics.Init()
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8542", nil)

	server := service.NewServer(config)

	log.Fatalf("serve: %s", server.ListenAndServe(":"+config.ListenPort))
}
