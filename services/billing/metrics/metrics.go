package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const serviceName = "billing"

var (
	ClientLag                 *prometheus.GaugeVec
	DBReadErrorsCount         *prometheus.CounterVec
	DBWriteErrorsCount        *prometheus.CounterVec
	StartTXErrorsCount        *prometheus.CounterVec
	CommitTXErrorsCount       *prometheus.CounterVec
	ProcessPaymentErrorsCount *prometheus.CounterVec
)

func Init() {
	ClientLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_client_lag",
			Help: "Lag of Kafka consumer per topic and partition",
		},
		[]string{"service", "topic", "partition"},
	)
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
	StartTXErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_start_tx_errors_total",
			Help: "Total number of errors encountered while starting a database transaction",
		},
		[]string{"service"},
	)
	CommitTXErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_commit_tx_errors_total",
			Help: "Total number of errors encountered while committing a database transaction",
		},
		[]string{"service"},
	)
	ProcessPaymentErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "process_payment_errors_total",
			Help: "Total number of errors encountered while processing payments",
		},
		[]string{"service"},
	)
	prometheus.MustRegister(ClientLag, DBWriteErrorsCount, DBReadErrorsCount, StartTXErrorsCount, CommitTXErrorsCount, ProcessPaymentErrorsCount)
}

func SetTopicPartitionLag(topic string, partition int32, lag int64) {
	ClientLag.WithLabelValues(serviceName, topic, fmt.Sprintf("%d", partition)).Set(float64(lag))
}

func IncDBReadErrorsCount() {
	DBReadErrorsCount.WithLabelValues(serviceName).Inc()
}

func IncDBWriteErrorsCount() {
	DBWriteErrorsCount.WithLabelValues(serviceName).Inc()
}

func IncStartTXErrorsCount() {
	StartTXErrorsCount.WithLabelValues(serviceName).Inc()
}

func IncCommitTXErrorsCount() {
	CommitTXErrorsCount.WithLabelValues(serviceName).Inc()
}

func IncProcessPaymentErrorsCount() {
	ProcessPaymentErrorsCount.WithLabelValues(serviceName).Inc()
}
