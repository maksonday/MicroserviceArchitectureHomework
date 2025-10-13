package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DBReadErrorsCount  prometheus.Counter
	DBWriteErrorsCount prometheus.Counter
)

func Init() {
	DBReadErrorsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "db_read_errors_total",
			Help: "Total number of errors encountered while reading from the database",
		},
	)
	DBWriteErrorsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "db_write_errors_total",
			Help: "Total number of errors encountered while writing to the database",
		},
	)
	prometheus.MustRegister(DBWriteErrorsCount)
	prometheus.MustRegister(DBReadErrorsCount)
}
