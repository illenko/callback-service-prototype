package callback

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/message"
	"github.com/segmentio/kafka-go"
)

const (
	defaultPollingIntervalMs   = 500
	defaultFetchSize           = 200
	defaultRetryPublishDelayMs = 10_000
	defaultMaxPublishAttempts  = 3
)

type Producer struct {
	repo               *db.CallbackRepository
	writer             *kafka.Writer
	pollingInterval    time.Duration
	fetchSize          int
	retryDelay         time.Duration
	maxPublishAttempts int
}

func NewProducer(repo *db.CallbackRepository, writer *kafka.Writer) *Producer {
	return &Producer{
		repo:               repo,
		writer:             writer,
		pollingInterval:    time.Duration(config.GetInt("CALLBACK_POLLING_INTERVAL_MS", defaultPollingIntervalMs)) * time.Millisecond,
		fetchSize:          config.GetInt("CALLBACK_FETCH_SIZE", defaultFetchSize),
		retryDelay:         time.Duration(config.GetInt("CALLBACK_RETRY_PUBLISH_DELAY_MS", defaultRetryPublishDelayMs)) * time.Millisecond,
		maxPublishAttempts: config.GetInt("MAX_PUBLISH_ATTEMPTS", defaultMaxPublishAttempts),
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
			log.Println("Context done, stopping producer")
			return
		}
	}
}

func (p *Producer) process(ctx context.Context) {
	log.Println("Starting transaction")
	tx, err := p.repo.BeginTx(ctx)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return
	}

	log.Println("Fetching unprocessed callbacks")
	callbacks, err := p.repo.GetUnprocessedCallbacks(ctx, tx, p.fetchSize)
	if err != nil {
		log.Printf("Error fetching unprocessed callbacks: %v", err)
		tx.Rollback(ctx)
		return
	}

	if len(callbacks) == 0 {
		log.Println("No unprocessed callbacks found")
		tx.Commit(ctx)
		return
	}

	var kafkaMessages []kafka.Message

	for _, entity := range callbacks {
		log.Printf("Preparing Kafka message for callback ID %s", entity.ID)

		callbackMessage := message.Callback{
			ID:        entity.ID,
			PaymentID: entity.PaymentID,
			Url:       entity.Url,
			Payload:   entity.Payload,
			Attempts:  entity.DeliveryAttempts,
		}

		// todo: add error handling
		messageBytes, _ := json.Marshal(callbackMessage)

		msg := kafka.Message{
			Key:   []byte(entity.PaymentID.String()), // Use payment ID as key to ensure ordering
			Value: messageBytes,
		}

		kafkaMessages = append(kafkaMessages, msg)
	}

	log.Println("Writing messages to Kafka")
	err = p.writer.WriteMessages(ctx, kafkaMessages...)
	if err != nil {
		log.Printf("Error writing messages to Kafka: %v", err)
	}

	for _, callback := range callbacks {
		log.Printf("Clear scheduled_at for callback ID %s", callback.ID)

		callback.PublishAttempts++

		if err != nil {
			// failed to post to Kafka
			errMsg := err.Error()
			callback.Error = &errMsg

			if callback.PublishAttempts >= 3 {
				log.Printf("Max attempts reached for callback ID %s", callback.ID)
				callback.ScheduledAt = nil
			} else {
				// schedule for retry
				scheduledAt := time.Now().Add(time.Duration(callback.PublishAttempts) * 10 * time.Second)
				callback.ScheduledAt = &scheduledAt
			}
		} else {
			callback.ScheduledAt = nil
			callback.Error = nil
		}

		err := p.repo.Update(ctx, tx, callback)

		if err != nil {
			log.Printf("Error updating scheduled_at for callback ID %s: %v", callback.ID, err)
			tx.Rollback(ctx)
			return
		} else {
			log.Printf("Successfully updated scheduled_at for callback ID %s", callback.ID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error committing transaction: %v", err)
		tx.Rollback(ctx)
	} else {
		log.Println("Transaction committed successfully")
	}
}
