package service

import (
	"context"
	"encoding/json"
	"order/config"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

var (
	notificationsProcessorOnce sync.Once
	notificationsProcessor     *NotificationsProcessor
)

type NotificationsProcessor struct {
	producer     sarama.AsyncProducer
	produceTopic string

	queuedMessages chan *NotificationMessage
}

type NotificationMessage struct {
	UserID  int64  `json:"user_id"`
	OrderID int64  `json:"order_id"`
	Message string `json:"message"`
}

func NewNotificationsProcessor(config *config.Config) {
	notificationsProcessorOnce.Do(func() {
		pConfig := sarama.NewConfig()
		version, err := sarama.ParseKafkaVersion(config.NotificationsProducerConfig.Version)
		if err != nil {
			zap.L().Fatal("failed to parse kafka version", zap.Error(err))
		}
		pConfig.Version = version
		pConfig.Net.TLS.Enable = false

		p, err := sarama.NewAsyncProducer(config.NotificationsProducerConfig.Brokers, pConfig)
		if err != nil {
			zap.L().Fatal("failed to start producer", zap.Error(err))
		}

		notificationsProcessor = &NotificationsProcessor{
			producer:       p,
			produceTopic:   config.NotificationsProducerConfig.Topic,
			queuedMessages: make(chan *NotificationMessage, 256),
		}
	})
}

func GetNotificationsProcessor() *NotificationsProcessor {
	return notificationsProcessor
}

func (p *NotificationsProcessor) AddMessage(msg *NotificationMessage) {
	p.queuedMessages <- msg
}

func (p *NotificationsProcessor) Run() {
	zap.L().Info("notifications processor started")

	ctx, cancel := context.WithCancel(context.Background())

	keepRunning := true

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range p.producer.Errors() {
			zap.L().Error("failed to produce message", zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

	ProducerLoop:
		for {
			select {
			case msg := <-p.queuedMessages:
				bytes, err := json.Marshal(msg)
				if err != nil {
					zap.L().Error("failed to marshal notification message", zap.Error(err))
					continue
				}
				zap.L().Sugar().Infof("producing message: %s", string(bytes))
				p.producer.Input() <- &sarama.ProducerMessage{Topic: p.produceTopic, Value: sarama.StringEncoder(string(bytes))}
			case <-ctx.Done():
				p.producer.AsyncClose() // Trigger a shutdown of the producer.
				break ProducerLoop
			}
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	for keepRunning {
		select {
		case <-ctx.Done():
			zap.L().Info("terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			zap.L().Info("terminating: via signal")
			keepRunning = false
		}
	}

	cancel()
	wg.Wait()
}
