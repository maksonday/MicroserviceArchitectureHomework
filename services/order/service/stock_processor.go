package service

import (
	"context"
	"encoding/json"
	"errors"
	"order/config"
	"order/db"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

var (
	stockProcessorOnce sync.Once
	stockProcessor     *StockProcessor
)

type StockProcessor struct {
	consumer     sarama.ConsumerGroup
	consumeTopic string

	producer     sarama.AsyncProducer
	produceTopic string

	queuedMessages chan *StockChangeMessage
}

func NewStockProcessor(config *config.Config) {
	stockProcessorOnce.Do(func() {
		cConfig := sarama.NewConfig()
		version, err := sarama.ParseKafkaVersion(config.StockConsumerConfig.Version)
		if err != nil {
			zap.L().Fatal("failed to parse kafka version", zap.Error(err))
		}
		cConfig.Version = version
		cConfig.Net.TLS.Enable = false

		c, err := sarama.NewConsumerGroup(config.StockConsumerConfig.Brokers, config.StockConsumerConfig.GroupID, cConfig)
		if err != nil {
			zap.L().Fatal("failed to start consumer", zap.Error(err))
		}

		pConfig := sarama.NewConfig()
		version, err = sarama.ParseKafkaVersion(config.StockProducerConfig.Version)
		if err != nil {
			zap.L().Fatal("failed to parse kafka version", zap.Error(err))
		}
		pConfig.Version = version
		pConfig.Net.TLS.Enable = false

		p, err := sarama.NewAsyncProducer(config.StockProducerConfig.Brokers, pConfig)
		if err != nil {
			zap.L().Fatal("failed to start producer", zap.Error(err))
		}

		stockProcessor = &StockProcessor{
			consumer:       c,
			consumeTopic:   config.StockConsumerConfig.Topic,
			producer:       p,
			produceTopic:   config.StockProducerConfig.Topic,
			queuedMessages: make(chan *StockChangeMessage, 256),
		}
	})
}

func GetStockProcessor() *StockProcessor {
	return stockProcessor
}

func (p *StockProcessor) AddMessage(msg *StockChangeMessage) {
	p.queuedMessages <- msg
}

func (p *StockProcessor) Run() {
	zap.L().Info("stock processor started")

	ctx, cancel := context.WithCancel(context.Background())

	keepRunning := true

	consumer := StockConsumer{
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
					zap.L().Error("failed to marshal stock message", zap.Error(err))
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

// StockConsumer represents a Sarama consumer group consumer
type StockConsumer struct {
	ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *StockConsumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *StockConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

const (
	StockChangeStatusPending int8 = iota
	StockChangeStatusOK
	StockChangeStatusFailed
)

const (
	StockAdd int8 = iota
	StockRemove
)

type StockChangeMessage struct {
	StockChangeIDs []int64 `json:"stock_change_ids"`
	OrderID        int64   `json:"order_id"`
	Action         int8    `json:"action"`
	Status         int8    `json:"status"` // 0 - pending, 1 - ok, 2 - failed
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (consumer *StockConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
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
			if err := consumer.processStock(message.Value); err != nil {
				zap.L().Error("failed to process stock message", zap.Error(err))
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

func (consumer *StockConsumer) processStock(data []byte) error {
	var msg StockChangeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	if msg.Status == StockChangeStatusPending || msg.OrderID == 0 || len(msg.StockChangeIDs) == 0 {
		zap.L().Warn("received bad stock_change message",
			zap.Int64s("stock_change_ids", msg.StockChangeIDs),
			zap.Int8("status", msg.Status))
		return nil
	}

	defer func() {
		zap.L().Sugar().Infof("processed stock message: %+v", msg)
	}()

	switch msg.Status {
	// успешно применили изменения на складе
	case StockChangeStatusOK:
		switch msg.Action {
		// зарезервировали товары на складе, создаем платеж
		case StockRemove:
			paymentID, err := db.CreatePayment(msg.OrderID, msg.StockChangeIDs)
			if err != nil {
				msg.Action = StockAdd
				GetStockProcessor().AddMessage(&msg)
			}
			GetPaymentsProcessor().AddMessage(&PaymentMessage{
				OrderID:        msg.OrderID,
				StockChangeIDs: msg.StockChangeIDs,
				PaymentID:      paymentID,
				Status:         PaymentStatusPending,
			})
			// что-то далее по цепочке пошло не так после резерва, отменяем заказ
		case StockAdd:
			db.RejectOrder(msg.OrderID)
		}
		// не удалось применить изменения на складе, отменяем заказ
	case StockChangeStatusFailed:
		db.RejectOrder(msg.OrderID)
	default:
		zap.L().Sugar().Errorf("unknown stock_change msg status: %d", msg.Status)
	}

	return nil
}
