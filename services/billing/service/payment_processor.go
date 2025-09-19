package service

import (
	"billing/config"
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type PaymentsProcessor struct {
	consumer     sarama.ConsumerGroup
	consumeTopic string

	producer     sarama.AsyncProducer
	produceTopic string
}

func NewPaymentsProcessor(config *config.Config) *PaymentsProcessor {
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

	pConfig := sarama.NewConfig()
	version, err = sarama.ParseKafkaVersion(config.ProducerConfig.Version)
	if err != nil {
		zap.L().Fatal("failed to parse kafka version", zap.Error(err))
	}
	pConfig.Version = version
	pConfig.Net.TLS.Enable = false

	p, err := sarama.NewAsyncProducer(config.ProducerConfig.Brokers, pConfig)
	if err != nil {
		zap.L().Fatal("failed to start producer", zap.Error(err))
	}

	return &PaymentsProcessor{
		consumer:     c,
		consumeTopic: config.ConsumerConfig.Topic,
		producer:     p,
		produceTopic: config.ProducerConfig.Topic,
	}
}

func (p *PaymentsProcessor) Run() {
	zap.L().Info("payment processor started")

	ctx, cancel := context.WithCancel(context.Background())

	keepRunning := true
	consumptionIsPaused := false

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
				zap.L().Fatal("Error from consumer: " + err.Error())
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

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

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
		case <-sigusr1:
			toggleConsumptionFlow(p.consumer, &consumptionIsPaused)
		}
	}
	cancel()
	wg.Wait()
	if err := p.consumer.Close(); err != nil {
		zap.L().Fatal("Error closing consumer: " + err.Error())
	}
}

func toggleConsumptionFlow(consumer sarama.ConsumerGroup, isPaused *bool) {
	if *isPaused {
		consumer.ResumeAll()
		zap.L().Info("Resuming consumption")
	} else {
		consumer.PauseAll()
		zap.L().Info("Pausing consumption")
	}

	*isPaused = !*isPaused
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
			zap.L().Sugar().Infof("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			session.MarkMessage(message, "")
		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}
