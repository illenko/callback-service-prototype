package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/logging"
	"callback-service/internal/message"
	"github.com/VictoriaMetrics/metrics"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"
)

var (
	// producer batch metrics
	producerErrorFetchingCounter = metrics.GetOrCreateCounter(`callback_producer_total{result="fetching_failed"}`)
	producerErrorKafkaCounter    = metrics.GetOrCreateCounter(`callback_producer_total{result="publish_failed"}`)
	producerErrorUpdateCounter   = metrics.GetOrCreateCounter(`callback_producer_total{result="db_update_failed"}`)
	producerSuccessCounter       = metrics.GetOrCreateCounter(`callback_producer_total{result="success"}`)

	producerProcessDurationHistogram = metrics.GetOrCreateHistogram(`callback_producer_duration_milliseconds`)

	// producer per message metrics
	producerMessagesPublishedCounter   = metrics.GetOrCreateCounter(`callback_producer_messages_total{result="published"}`)
	producerMessagesMaxAttemptsCounter = metrics.GetOrCreateCounter(`callback_producer_messages_total{result="max_attempts_reached"}`)
	producerMessagesRescheduledCounter = metrics.GetOrCreateCounter(`callback_producer_messages_total{result="rescheduled"}`)
)

type Producer struct {
	repo               *db.CallbackRepository
	writer             *kafka.Writer
	pollingInterval    time.Duration
	fetchSize          int
	retryDelay         time.Duration
	maxPublishAttempts int
	logger             *slog.Logger
}

func NewProducer(repo *db.CallbackRepository, writer *kafka.Writer, cfg config.CallbackProducer, logger *slog.Logger) *Producer {
	return &Producer{
		repo:               repo,
		writer:             writer,
		pollingInterval:    time.Duration(cfg.PollingIntervalMs) * time.Millisecond,
		fetchSize:          cfg.FetchSize,
		retryDelay:         time.Duration(cfg.RescheduleDelayMs) * time.Millisecond,
		maxPublishAttempts: cfg.MaxPublishAttempts,
		logger:             logger.With("component", "callback.producer"),
	}
}

func (p *Producer) Start(ctx context.Context) {
	ticker := time.NewTicker(p.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.process(ctx)
		case <-ctx.Done():
			p.logger.InfoContext(ctx, "Context done, stopping producer")
			return
		}
	}
}

func (p *Producer) process(ctx context.Context) {
	startTime := time.Now()

	ctx = logging.AppendCtx(ctx, slog.String("correlationId", uuid.New().String()))

	p.logger.DebugContext(ctx, "Starting DB transaction")
	tx, err := p.repo.BeginTx(ctx)
	if err != nil {
		p.logger.ErrorContext(ctx, fmt.Sprintf("Error starting transaction: %v", err))
		producerErrorFetchingCounter.Inc()
		return
	}

	defer tx.Rollback(ctx)

	p.logger.DebugContext(ctx, "Fetching unprocessed callbacks")

	callbacks, err := p.repo.GetUnprocessedCallbacks(ctx, tx, p.fetchSize)
	if err != nil {
		p.logger.ErrorContext(ctx, fmt.Sprintf("Error fetching callbacks: %v", err))
		producerErrorFetchingCounter.Inc()
		return
	}

	if len(callbacks) == 0 {
		p.logger.InfoContext(ctx, "No unprocessed callbacks found")
		producerSuccessCounter.Inc()
		return
	} else {
		kafkaMessages := p.toKafkaMessages(callbacks)

		p.logger.InfoContext(ctx, "Writing messages to Kafka", slog.Int("messageCount", len(kafkaMessages)))

		err = p.writer.WriteMessages(ctx, kafkaMessages...)

		if err != nil {
			p.logger.ErrorContext(ctx, fmt.Sprintf("Error writing messages to Kafka: %v", err))
			producerErrorKafkaCounter.Inc()
		}

		p.updateCallbacks(ctx, tx, callbacks, err)
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.ErrorContext(ctx, fmt.Sprintf("Error committing transaction: %v", err))

		producerErrorUpdateCounter.Inc()
	} else {
		p.logger.InfoContext(ctx, "Transaction committed successfully")

		producerSuccessCounter.Inc()
	}

	producerProcessDurationHistogram.Update(float64(time.Since(startTime).Milliseconds()))

}

func (p *Producer) toKafkaMessages(callbacks []*db.CallbackMessageEntity) []kafka.Message {
	var kafkaMessages []kafka.Message

	for _, entity := range callbacks {

		callbackMessage := message.Callback{
			ID:        entity.ID,
			PaymentID: entity.PaymentID,
			Url:       entity.Url,
			Payload:   entity.Payload,
			Attempts:  entity.DeliveryAttempts,
		}

		messageBytes, _ := json.Marshal(callbackMessage)

		msg := kafka.Message{
			Key:   []byte(entity.PaymentID.String()), // Use payment ID as key to ensure ordering
			Value: messageBytes,
		}

		kafkaMessages = append(kafkaMessages, msg)
	}
	return kafkaMessages
}

func (p *Producer) updateCallbacks(ctx context.Context, tx pgx.Tx, callbacks []*db.CallbackMessageEntity, kafkaErr error) {
	for _, callback := range callbacks {
		messageCtx := logging.AppendCtx(ctx, slog.String("callbackId", callback.ID.String()))
		p.logger.InfoContext(messageCtx, "Update callback message values")

		callback.PublishAttempts++
		if kafkaErr != nil {
			errMsg := kafkaErr.Error()
			callback.Error = &errMsg
			p.handleMaxAttempts(messageCtx, callback)
		} else {
			callback.ScheduledAt = nil
			callback.Error = nil
			producerMessagesPublishedCounter.Inc()
		}

		if err := p.repo.Update(messageCtx, tx, callback); err != nil {
			p.logger.ErrorContext(messageCtx, fmt.Sprintf("Error updating callback entity: %v", err))
			return
		}
		p.logger.InfoContext(messageCtx, fmt.Sprintf("Callback entity updated successfully: %v", callback))
	}
}

func (p *Producer) handleMaxAttempts(ctx context.Context, callback *db.CallbackMessageEntity) {
	if callback.PublishAttempts >= p.maxPublishAttempts {
		p.logger.WarnContext(ctx, "Max attempts reached for callback")
		callback.ScheduledAt = nil
		producerMessagesMaxAttemptsCounter.Inc()
	} else {
		scheduledAt := time.Now().Add(time.Duration(callback.PublishAttempts) * p.retryDelay)
		callback.ScheduledAt = &scheduledAt
		producerMessagesRescheduledCounter.Inc()
	}
}
