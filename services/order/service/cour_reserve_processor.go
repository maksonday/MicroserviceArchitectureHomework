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
	courReserveProcessorOnce sync.Once
	courReserveProcessor     *CourReserveProcessor

	courReserveRetryCount int
)

type CourReserveProcessor struct {
	consumer     sarama.ConsumerGroup
	consumeTopic string

	producer     sarama.AsyncProducer
	produceTopic string

	queuedMessages chan *CourReserveMessage
}

func NewCourReserveProcessor(config *config.Config) {
	courReserveProcessorOnce.Do(func() {
		cConfig := sarama.NewConfig()
		version, err := sarama.ParseKafkaVersion(config.CourReserveConsumerConfig.Version)
		if err != nil {
			zap.L().Fatal("failed to parse kafka version", zap.Error(err))
		}
		cConfig.Version = version
		cConfig.Net.TLS.Enable = false

		c, err := sarama.NewConsumerGroup(config.CourReserveConsumerConfig.Brokers, config.CourReserveConsumerConfig.GroupID, cConfig)
		if err != nil {
			zap.L().Fatal("failed to start consumer", zap.Error(err))
		}

		pConfig := sarama.NewConfig()
		version, err = sarama.ParseKafkaVersion(config.CourReserveProducerConfig.Version)
		if err != nil {
			zap.L().Fatal("failed to parse kafka version", zap.Error(err))
		}
		pConfig.Version = version
		pConfig.Net.TLS.Enable = false

		p, err := sarama.NewAsyncProducer(config.CourReserveProducerConfig.Brokers, pConfig)
		if err != nil {
			zap.L().Fatal("failed to start producer", zap.Error(err))
		}

		courReserveProcessor = &CourReserveProcessor{
			consumer:       c,
			consumeTopic:   config.CourReserveConsumerConfig.Topic,
			producer:       p,
			produceTopic:   config.CourReserveProducerConfig.Topic,
			queuedMessages: make(chan *CourReserveMessage, 256),
		}

		courReserveRetryCount = config.CourReserveRetryCount
	})
}

func GetCourReserveProcessor() *CourReserveProcessor {
	return courReserveProcessor
}

func (p *CourReserveProcessor) AddMessage(msg *CourReserveMessage) {
	p.queuedMessages <- msg
}

func (p *CourReserveProcessor) Run() {
	zap.L().Info("cour_reserve processor started")

	ctx, cancel := context.WithCancel(context.Background())

	keepRunning := true

	consumer := CourReserveConsumer{
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
					zap.L().Error("failed to marshal cour_reserve message", zap.Error(err))
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

// CourReserveConsumer represents a Sarama consumer group consumer
type CourReserveConsumer struct {
	ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *CourReserveConsumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *CourReserveConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

const (
	CourReserveStatusPending int8 = iota
	CourReserveStatusOK
	CourReserveStatusFailed
)

const (
	RevertCourReserve = iota
	CourReserve
)

type CourReserveMessage struct {
	PaymentID         int64   `json:"payment_id"`
	OrderID           int64   `json:"order_id"`
	StockChangeIDs    []int64 `json:"stock_change_ids"`
	CourReservationID int64   `json:"cour_reservation_id"`
	Action            int8    `json:"action"`
	Status            int8    `json:"status"` // 0 - pending, 1 - ok, 2 - failed
	RetryCount        int     `json:"retry_count"`
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (consumer *CourReserveConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
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
			if err := consumer.processCourReserve(message.Value); err != nil {
				zap.L().Error("failed to process cour_reserve message", zap.Error(err))
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

func (consumer *CourReserveConsumer) processCourReserve(data []byte) error {
	var msg CourReserveMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	if msg.Status == CourReserveStatusPending || msg.CourReservationID == 0 {
		zap.L().Warn("received bad cour_reserve message",
			zap.Int64("cour_reserve_id", msg.CourReservationID),
			zap.Int8("status", msg.Status))
		return nil
	}

	defer func() {
		zap.L().Sugar().Infof("processed cour_reserve message: %+v", msg)
	}()

	switch msg.Status {
	case CourReserveStatusOK:
		switch msg.Action {
		case CourReserve:
			// подтверждаем заказ и отправляем уведомление на почту
			db.OrderSetStatus(msg.OrderID, "delivery")
			go NotifyUser(msg.OrderID, OrderStatusDelivery)
		case RevertCourReserve:
			// что-то пошло не так, освободили слот курьеру, возвращаем деньги клиенту
			// заказ отменится по цепочке после роллбека резерва слота
			newCourReserveID, err := db.RevertCourReserve(msg.CourReservationID)
			if err != nil {
				zap.L().Error("failed to revert cour_reserve", zap.Error(err))
				return nil
			}

			GetCourReserveProcessor().AddMessage(&CourReserveMessage{
				StockChangeIDs:    msg.StockChangeIDs,
				OrderID:           msg.OrderID,
				Action:            RevertCourReserve,
				Status:            CourReserveStatusPending,
				CourReservationID: newCourReserveID,
			})
		}
	case CourReserveStatusFailed:
		// ретраим
		if msg.RetryCount < courReserveRetryCount {
			courReserveID, err := db.CreateCourReserve(msg.OrderID)
			if err != nil {
				zap.L().Error("create cour_reserve error", zap.Error(err))
				return nil
			}

			GetCourReserveProcessor().AddMessage(&CourReserveMessage{
				OrderID:           msg.OrderID,
				StockChangeIDs:    msg.StockChangeIDs,
				PaymentID:         msg.PaymentID,
				Status:            CourReserveStatusPending,
				Action:            CourReserve,
				CourReservationID: courReserveID,
				RetryCount:        msg.RetryCount + 1,
			})
			return nil
		}

		// что-то пошло не так, все попытки повторить резерв курьера исчерпаны
		// возвращаем деньги, затем возвращаем товары на склад
		// заказ отменится по цепочке после роллбека склада
		newPaymentID, err := db.RevertPayment(msg.PaymentID)
		if err != nil {
			zap.L().Error("failed to revert payment", zap.Error(err))
			return nil
		}

		GetPaymentsProcessor().AddMessage(&PaymentMessage{
			StockChangeIDs: msg.StockChangeIDs,
			OrderID:        msg.OrderID,
			Action:         Deposit,
			Status:         PaymentStatusPending,
			PaymentID:      newPaymentID,
		})
	default:
		zap.L().Sugar().Errorf("unknown cour_reserve msg status: %d", msg.Status)
	}

	return nil
}
