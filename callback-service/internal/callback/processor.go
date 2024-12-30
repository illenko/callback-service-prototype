package callback

import (
	"context"
	"log"

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

func (p *Processor) Process(ctx context.Context, message message.Callback) error {
	log.Printf("Processing message: %+v", message)

	p.sem <- struct{}{}
	go func() {
		defer func() { <-p.sem }()

		err := p.sender.Send(ctx, message.Url, message.Payload)
		if err != nil {
			log.Printf("Error sending callback: %v", err)
		} else {
			log.Printf("Successfully processed and sent event with ID: %s", message.ID)
		}
	}()

	return nil
}
