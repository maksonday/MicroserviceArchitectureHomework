package service

import (
	"billing/config"
	"billing/db"
	"context"
	"database/sql"
	"encoding/json"
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

	consumer := Consumer{
		ready:             make(chan bool),
		processedMessages: make(chan *PaymentMessage, 256),
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
			case msg := <-consumer.processedMessages:
				bytes, err := json.Marshal(msg)
				if err != nil {
					zap.L().Error("failed to marshal payment message", zap.Error(err))
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
	if err := p.consumer.Close(); err != nil {
		zap.L().Fatal("error closing consumer: " + err.Error())
	}
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ready             chan bool
	processedMessages chan *PaymentMessage
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

const (
	PaymentStatusPending int8 = iota
	PaymentStatusOK
	PaymentStatusFailed
)

const (
	Deposit = iota
	Pay
)

type PaymentMessage struct {
	PaymentID      int64   `json:"payment_id"`
	OrderID        int64   `json:"order_id"`
	StockChangeIDs []int64 `json:"stock_change_ids"`
	Action         int8    `json:"action"`
	Status         int8    `json:"status"` // 0 - pending, 1 - ok, 2 - failed
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
			if err := consumer.processPayment(message.Value); err != nil {
				zap.L().Error("failed to process payment message", zap.Error(err))
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

func (consumer *Consumer) processPayment(data []byte) error {
	var msg PaymentMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	if msg.Status != PaymentStatusPending || msg.PaymentID == 0 || msg.OrderID == 0 {
		zap.L().Warn("received bad payment message",
			zap.Int64("payment_id", msg.PaymentID),
			zap.Int64("order_id", msg.OrderID),
			zap.Int8("status", msg.Status))
		return nil
	}

	produce := func(msg *PaymentMessage) {
		zap.L().Sugar().Infof("processed payment message: %+v", *msg)
		consumer.processedMessages <- msg
	}

	switch msg.Action {
	case Deposit, Pay:
		if err := db.ProcessPayment(msg.PaymentID, msg.Action); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}

			db.RejectPayment(msg.PaymentID, err.Error())

			zap.L().Error(
				"failed to process payment",
				zap.Error(err),
				zap.Int64("payment_id", msg.PaymentID),
				zap.Int64("order_id", msg.OrderID),
				zap.Int8("action", msg.Action),
				zap.Int8("status", msg.Status),
				zap.Int64s("stock_change_ids", msg.StockChangeIDs),
			)
			msg.Status = PaymentStatusFailed
			produce(&msg)
			return nil
		}
		db.ApprovePayment(msg.PaymentID)
		msg.Status = PaymentStatusOK
		produce(&msg)
		return nil
	default:
		return nil
	}
}
