package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const serviceName = "users"

var (
	DBReadErrorsCount  *prometheus.CounterVec
	DBWriteErrorsCount *prometheus.CounterVec
)

func Init() {
	DBReadErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_read_errors_total",
			Help: "Total number of errors encountered while reading from the database",
		},
		[]string{"service"},
	)
	DBWriteErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_write_errors_total",
			Help: "Total number of errors encountered while writing to the database",
		},
		[]string{"service"},
	)
	prometheus.MustRegister(DBWriteErrorsCount)
	prometheus.MustRegister(DBReadErrorsCount)
}

func IncDBReadErrorsCount() {
	DBReadErrorsCount.WithLabelValues(serviceName).Inc()
}

func IncDBWriteErrorsCount() {
	DBWriteErrorsCount.WithLabelValues(serviceName).Inc()
}
