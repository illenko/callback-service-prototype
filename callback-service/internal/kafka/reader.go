package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"callback-service/internal/callback"
	"callback-service/internal/event"
	"callback-service/internal/message"
	"github.com/VictoriaMetrics/metrics"
	"github.com/segmentio/kafka-go"
)

type Metrics struct {
	ReadErrorCounter      *metrics.Counter
	UnmarshalErrorCounter *metrics.Counter
	ProcessErrorCounter   *metrics.Counter
	SuccessCounter        *metrics.Counter
}

var (
	// Define metrics for payment events
	paymentEventMetrics = Metrics{
		ReadErrorCounter:      metrics.GetOrCreateCounter(`kafka_reader_total{result="read_error",type="payment_event"}`),
		UnmarshalErrorCounter: metrics.GetOrCreateCounter(`kafka_reader_total{result="unmarshal_error",type="payment_event"}`),
		ProcessErrorCounter:   metrics.GetOrCreateCounter(`kafka_reader_total{result="process_error",type="payment_event"}`),
		SuccessCounter:        metrics.GetOrCreateCounter(`kafka_reader_total{result="success",type="payment_event"}`),
	}

	// Define metrics for callback messages
	callbackMessageMetrics = Metrics{
		ReadErrorCounter:      metrics.GetOrCreateCounter(`kafka_reader_total{result="read_error",type="callback_message"}`),
		UnmarshalErrorCounter: metrics.GetOrCreateCounter(`kafka_reader_total{result="unmarshal_error",type="callback_message"}`),
		ProcessErrorCounter:   metrics.GetOrCreateCounter(`kafka_reader_total{result="process_error",type="callback_message"}`),
		SuccessCounter:        metrics.GetOrCreateCounter(`kafka_reader_total{result="success",type="callback_message"}`),
	}
)

func NewReader(kafkaURL, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(kafkaURL, ","),
		GroupID: groupID,
		Topic:   topic,
	})
}

func ReadPaymentEvents(reader *kafka.Reader, processor *event.Processor, logger *slog.Logger) {
	readMessages(context.Background(), reader, logger, func(ctx context.Context, value []byte) error {
		var e message.PaymentEvent
		if err := json.Unmarshal(value, &e); err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error unmarshalling message: %v", err))
			paymentEventMetrics.UnmarshalErrorCounter.Inc()
			return err
		}
		return processor.Process(ctx, e)
	}, paymentEventMetrics)
}

func ReadCallbackMessages(reader *kafka.Reader, processor *callback.Processor, logger *slog.Logger) {
	readMessages(context.Background(), reader, logger, func(ctx context.Context, value []byte) error {
		var c message.Callback
		if err := json.Unmarshal(value, &c); err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error unmarshalling message: %v", err))
			callbackMessageMetrics.UnmarshalErrorCounter.Inc()
			return err
		}
		return processor.Process(ctx, c)
	}, callbackMessageMetrics)
}

func readMessages(ctx context.Context, reader *kafka.Reader, logger *slog.Logger, process func(context.Context, []byte) error, kafkaMetrics Metrics) {
	go func() {
		for {
			logger.InfoContext(ctx, "Waiting for messages from Kafka...")
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("Error reading message: %v", err))
				kafkaMetrics.ReadErrorCounter.Inc()
				continue
			}
			logger.InfoContext(ctx, fmt.Sprintf("Received message: %s from topic %s", string(m.Value), m.Topic))

			err = process(ctx, m.Value)
			if err != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("Error processing message: %v", err))
				kafkaMetrics.ProcessErrorCounter.Inc()
				continue
			}
			kafkaMetrics.SuccessCounter.Inc()
		}
	}()
}
