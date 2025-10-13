package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ClientLag                 *prometheus.GaugeVec
	DBReadErrorsCount         prometheus.Counter
	DBWriteErrorsCount        prometheus.Counter
	StartTXErrorsCount        prometheus.Counter
	CommitTXErrorsCount       prometheus.Counter
	ProcessPaymentErrorsCount prometheus.Counter
)

func Init() {
	ClientLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_client_lag",
			Help: "Lag of Kafka consumer per topic and partition",
		},
		[]string{"topic", "partition"},
	)
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
	StartTXErrorsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "db_start_tx_errors_total",
			Help: "Total number of errors encountered while starting a database transaction",
		},
	)
	CommitTXErrorsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "db_commit_tx_errors_total",
			Help: "Total number of errors encountered while committing a database transaction",
		},
	)
	ProcessPaymentErrorsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "process_payment_errors_total",
			Help: "Total number of errors encountered while processing payments",
		},
	)

	prometheus.MustRegister(ClientLag, DBWriteErrorsCount, DBReadErrorsCount, StartTXErrorsCount, CommitTXErrorsCount, ProcessPaymentErrorsCount)
}

func SetTopicPartitionLag(topic string, partition int32, lag int64) {
	ClientLag.WithLabelValues(topic, fmt.Sprintf("%d", partition)).Set(float64(lag))
}
