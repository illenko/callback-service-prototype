package callback

import (
	"context"
	"log"
	"time"

	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/message"
)

const (
	DefaultParallelism = 1000
	DefaultMaxAttempts = 3
)

type Processor struct {
	repo        *db.CallbackRepository
	sender      *Sender
	sem         chan struct{}
	maxAttempts int
}

func NewCallbackProcessor(repo *db.CallbackRepository, sender *Sender) *Processor {
	parallelism := config.GetEnvInt("CALLBACK_PARALLELISM", DefaultParallelism)
	maxAttempts := config.GetEnvInt("MAX_DELIVERY_ATTEMPTS", DefaultMaxAttempts)

	return &Processor{
		repo:        repo,
		sender:      sender,
		sem:         make(chan struct{}, parallelism),
		maxAttempts: maxAttempts,
	}
}

func (p *Processor) Process(ctx context.Context, message message.Callback) error {
	log.Printf("Processing message: %+v", message)

	p.sem <- struct{}{}
	go func() {
		defer func() { <-p.sem }()

		tx, err := p.repo.BeginTx(ctx)
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			return
		}

		entity, err := p.repo.SelectForUpdateByID(ctx, tx, message.ID)
		if err != nil {
			log.Printf("Error selecting for update by ID: %v", err)
			tx.Rollback(ctx)
			return
		}

		err = p.sender.Send(ctx, message.Url, message.Payload)
		entity.DeliveryAttempts++

		if err != nil {
			log.Printf("Error sending callback: %v", err)

			// Check if we have reached the max number of attempts
			if entity.DeliveryAttempts >= p.maxAttempts {
				log.Printf("Max delivery attempts reached for callback ID: %s", message.ID)
				entity.ScheduledAt = nil
				errorMsg := "Max delivery attempts reached. " + err.Error()
				entity.Error = &errorMsg
			} else {
				scheduledAt := time.Now().Add(time.Duration(entity.DeliveryAttempts) * 10 * time.Second)
				errorMsg := err.Error()
				entity.ScheduledAt = &scheduledAt
				entity.Error = &errorMsg
			}

		} else {
			log.Printf("Successfully processed and sent event with ID: %s", message.ID)

			now := time.Now()

			entity.DeliveredAt = &now
			entity.ScheduledAt = nil
			entity.Error = nil
		}

		log.Printf("Updating callback with ID: %s", message.ID)

		err = p.repo.Update(ctx, tx, entity)
		if err != nil {
			log.Printf("Error updating callback: %v", err)
			tx.Rollback(ctx)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("Error committing transaction: %v", err)
			tx.Rollback(ctx)
		} else {
			log.Println("Transaction committed successfully")
		}
	}()

	return nil
}
