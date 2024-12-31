package callback

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"callback-service/internal/db"
	"callback-service/internal/message"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	repo   *db.CallbackRepository
	writer *kafka.Writer
}

func NewProducer(repo *db.CallbackRepository, writer *kafka.Writer) *Producer {
	return &Producer{repo: repo, writer: writer}
}

func (p *Producer) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
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
	callbacks, err := p.repo.GetUnprocessedCallbacks(ctx, tx, 100)
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
			Attempts:  entity.Attempts,
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

		entity := &db.CallbackMessageEntity{
			ID:          callback.ID,
			ScheduledAt: nil,
			Error:       nil,
		}

		if err != nil {
			errMsg := err.Error()
			entity.Error = &errMsg
		} else {
			entity.Error = nil
		}

		err := p.repo.ClearScheduledAt(ctx, tx, entity)

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
