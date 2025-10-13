package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ClientLag *prometheus.GaugeVec
)

func Init() {
	ClientLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_client_lag",
			Help: "Lag of Kafka consumer per topic and partition",
		},
		[]string{"topic", "partition"},
	)
	prometheus.MustRegister(ClientLag)
}

func SetTopicPartitionLag(topic string, partition int32, lag int64) {
	ClientLag.WithLabelValues(topic, fmt.Sprintf("%d", partition)).Set(float64(lag))
}
