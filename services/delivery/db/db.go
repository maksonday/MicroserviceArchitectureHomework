package db

import (
	"database/sql"
	"delivery/config"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var (
	onceDB sync.Once
	conn   *sql.DB

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

func Init(config *config.DBConfig) error {
	var err error
	onceDB.Do(func() {
		conn, err = sql.Open("postgres", getConnStr(config))
		if err != nil {
			return
		}

		err = conn.Ping()

		retryCount = config.Retry.Count
		retryDelay = config.Retry.Delay
	})

	return err
}

func GetConn() *sql.DB {
	return conn
}
