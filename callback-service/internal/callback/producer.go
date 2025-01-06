package callback

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/logcontext"
	"callback-service/internal/message"
	"github.com/VictoriaMetrics/metrics"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	defaultPollingIntervalMs   = 500
	defaultFetchSize           = 200
	defaultRetryPublishDelayMs = 10_000
	defaultMaxPublishAttempts  = 3
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

func NewProducer(repo *db.CallbackRepository, writer *kafka.Writer, logger *slog.Logger) *Producer {
	return &Producer{
		repo:               repo,
		writer:             writer,
		pollingInterval:    time.Duration(config.GetInt("CALLBACK_POLLING_INTERVAL_MS", defaultPollingIntervalMs)) * time.Millisecond,
		fetchSize:          config.GetInt("CALLBACK_FETCH_SIZE", defaultFetchSize),
		retryDelay:         time.Duration(config.GetInt("CALLBACK_RETRY_PUBLISH_DELAY_MS", defaultRetryPublishDelayMs)) * time.Millisecond,
		maxPublishAttempts: config.GetInt("MAX_PUBLISH_ATTEMPTS", defaultMaxPublishAttempts),
		logger:             logger,
	}
}

func (p *Producer) Start(ctx context.Context) {
	go func() {
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
	}()
}

func (p *Producer) process(ctx context.Context) {
	startTime := time.Now()

	// set runId as a correlation id for all logs in scope
	ctx = logcontext.AppendCtx(ctx, slog.String("runId", uuid.New().String()))

	p.logger.InfoContext(ctx, "Starting transaction")
	tx, err := p.repo.BeginTx(ctx)
	if err != nil {
		p.logger.ErrorContext(ctx, "Error starting transaction", "error", err)
		producerErrorFetchingCounter.Inc()
		return
	}

	defer tx.Rollback(ctx)

	p.logger.InfoContext(ctx, "Fetching unprocessed callbacks")
	callbacks, err := p.repo.GetUnprocessedCallbacks(ctx, tx, p.fetchSize)
	if err != nil {
		p.logger.ErrorContext(ctx, "Error fetching unprocessed callbacks", "error", err)
		producerErrorFetchingCounter.Inc()
		return
	}

	if len(callbacks) == 0 {
		p.logger.InfoContext(ctx, "No unprocessed callbacks found")
		producerSuccessCounter.Inc()
		return
	} else {
		kafkaMessages := p.toKafkaMessages(ctx, callbacks)

		p.logger.InfoContext(ctx, "Writing messages to Kafka")

		err = p.writer.WriteMessages(ctx, kafkaMessages...)
		if err != nil {
			p.logger.ErrorContext(ctx, "Error writing messages to Kafka", "error", err)
			producerErrorKafkaCounter.Inc()
		}

		for _, callback := range callbacks {
			messageCtx := logcontext.AppendCtx(ctx, slog.String("id", callback.ID.String()))

			p.logger.InfoContext(messageCtx, "Update callback message values")

			callback.PublishAttempts++

			if err != nil {
				errMsg := err.Error()
				callback.Error = &errMsg

				if callback.PublishAttempts >= p.maxPublishAttempts {
					p.logger.WarnContext(messageCtx, "Max attempts reached for callback")
					callback.ScheduledAt = nil

					producerMessagesMaxAttemptsCounter.Inc()
				} else {
					scheduledAt := time.Now().Add(time.Duration(callback.PublishAttempts) * p.retryDelay)
					callback.ScheduledAt = &scheduledAt

					producerMessagesRescheduledCounter.Inc()
				}
			} else {
				callback.ScheduledAt = nil
				callback.Error = nil

				producerMessagesPublishedCounter.Inc()
			}

			err := p.repo.Update(messageCtx, tx, callback)
			if err != nil {
				p.logger.ErrorContext(messageCtx, "Error updating callback", "error", err)

				return
			} else {
				p.logger.InfoContext(messageCtx, "Updated callback entity values", "value", callback)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.ErrorContext(ctx, "Error committing transaction", "error", err)

		producerErrorUpdateCounter.Inc()
	} else {
		p.logger.InfoContext(ctx, "Transaction committed successfully")

		producerSuccessCounter.Inc()
	}

	producerProcessDurationHistogram.Update(float64(time.Since(startTime).Milliseconds()))

}

func (p *Producer) toKafkaMessages(ctx context.Context, callbacks []*db.CallbackMessageEntity) []kafka.Message {
	var kafkaMessages []kafka.Message

	for _, entity := range callbacks {
		p.logger.DebugContext(ctx, "Preparing Kafka message for callback ID", "id", entity.ID)

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
