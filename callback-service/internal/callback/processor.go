package callback

import (
	"context"
	"log"
	"time"

	"callback-service/internal/db"
	"callback-service/internal/message"
)

type Processor struct {
	repo   *db.CallbackRepository
	sender *Sender
	sem    chan struct{}
}

func NewCallbackProcessor(repo *db.CallbackRepository, sender *Sender) *Processor {
	return &Processor{
		repo:   repo,
		sender: sender,
		sem:    make(chan struct{}, 1000),
	}
}

const maxAttempts = 3

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
		attempts := entity.Attempts + 1

		if err != nil {
			log.Printf("Error sending callback: %v", err)
			err = p.repo.UpdateScheduledAtAndAttemptsByID(ctx, tx, message.ID, time.Now().Add(time.Duration(attempts)*10*time.Second), attempts, err.Error())
			if err != nil {
				log.Printf("Error updating scheduled_at and attempts: %v", err)
				tx.Rollback(ctx)
				return
			}
		} else {
			log.Printf("Successfully processed and sent event with ID: %s", message.ID)
			err = p.repo.UpdateAttemptsAndDeliveredAtByID(ctx, tx, message.ID, attempts, time.Now())
			if err != nil {
				log.Printf("Error updating delivered_at and attempts: %v", err)
				tx.Rollback(ctx)
				return
			}
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
