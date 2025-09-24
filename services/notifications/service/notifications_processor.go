package service

import (
	"context"
	"encoding/json"
	"errors"
	"notifications/config"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type NotificationsProcessor struct {
	consumer     sarama.ConsumerGroup
	consumeTopic string
}

func NewNotificationsProcessor(config *config.Config) *NotificationsProcessor {
	cConfig := sarama.NewConfig()
	version, err := sarama.ParseKafkaVersion(config.ConsumerConfig.Version)
	if err != nil {
		zap.L().Fatal("failed to parse kafka version", zap.Error(err))
	}
	cConfig.Version = version
	cConfig.Net.TLS.Enable = false

	c, err := sarama.NewConsumerGroup(config.ConsumerConfig.Brokers, config.ConsumerConfig.GroupID, cConfig)
	if err != nil {
		zap.L().Fatal("failed to start consumer", zap.Error(err))
	}

	return &NotificationsProcessor{
		consumer:     c,
		consumeTopic: config.ConsumerConfig.Topic,
	}
}

func (p *NotificationsProcessor) Run() {
	zap.L().Info("notifications processor started")

	ctx, cancel := context.WithCancel(context.Background())

	keepRunning := true

	consumer := Consumer{
		ready: make(chan bool),
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := p.consumer.Consume(ctx, []string{p.consumeTopic}, &consumer); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}
				zap.L().Fatal("error from consumer: " + err.Error())
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	zap.L().Info("Sarama consumer up and running!...")

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
	if err := p.consumer.Close(); err != nil {
		zap.L().Fatal("error closing consumer: " + err.Error())
	}
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

type NotificationMessage struct {
	UserID  int64  `json:"user_id"`
	Message string `json:"message"`
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/IBM/sarama/blob/main/consumer_group.go#L27-L29
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				zap.L().Info("message channel was closed")
				return nil
			}
			zap.L().Sugar().Infof("message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			if err := consumer.processNotification(message.Value); err != nil {
				zap.L().Error("failed to process notification message", zap.Error(err))
			}
			session.MarkMessage(message, "")
		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}

func (consumer *Consumer) processNotification(data []byte) error {
	var msg NotificationMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	if msg.UserID == 0 || len(msg.Message) == 0 {
		zap.L().Warn("received bad notification message",
			zap.Int64("user_id", msg.UserID),
			zap.String("message", msg.Message))
		return nil
	}

	return nil
}
