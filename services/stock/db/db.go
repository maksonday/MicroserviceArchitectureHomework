package db

import (
	"database/sql"
	"log"
	"os"
	"stock/config"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var (
	onceDB sync.Once
	master *sql.DB
	slave  *sql.DB

	retryCount uint64 = 3
	retryDelay        = time.Second
)

func getConnStr(config *config.DBConfig) string {
	return "host=" + config.Host +
		" port=" + strconv.Itoa(config.Port) +
		" user=" + config.User +
		" password=" + getPassword() +
		" dbname=" + config.Database +
		" sslmode=" + config.SSLMode
}

func getPassword() string {
	data, err := os.ReadFile("/secret/postgres/password")
	if err != nil {
		log.Fatalf("failed to read password file: %s", err)
	}

	return string(data)
}

func Init(config *config.Config) error {
	var err error
	onceDB.Do(func() {
		retryCount = config.DBRetry.Count
		retryDelay = config.DBRetry.Delay

		master, err = sql.Open("postgres", getConnStr(config.DBMaster))
		if err != nil {
			return
		}

		if err = master.Ping(); err != nil {
			return
		}

		slave, err = sql.Open("postgres", getConnStr(config.DBReplica))
		if err != nil {
			return
		}

		err = slave.Ping()
	})

	return err
}

func GetConn() *sql.DB {
	return master
}

func GetConnSlave() *sql.DB {
	return slave
}
