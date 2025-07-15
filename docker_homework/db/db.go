package db

import (
	"database/sql"
	"miniapp/config"
	"strconv"
	"sync"

	"github.com/go-sql-driver/mysql"
)

var (
	onceDB sync.Once
	conn   *sql.DB
)

func Init(config *config.DBConfig) error {
	var err error
	onceDB.Do(func() {
		conf := mysql.NewConfig()
		conf.User = config.User
		conf.Passwd = config.Password
		conf.Net = "tcp"
		conf.Addr = config.Host + ":" + strconv.Itoa(config.Port)
		conf.DBName = config.Database

		conn, err = sql.Open("mysql", conf.FormatDSN())
		if err != nil {
			return
		}

		err = conn.Ping()
	})

	return err
}

func GetConn() *sql.DB {
	return conn
}
