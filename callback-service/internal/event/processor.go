package event

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"callback-service/internal/db"
	"callback-service/internal/message"
	"callback-service/internal/payload"
)

type Processor struct {
	repo *db.CallbackRepository
}

func NewProcessor(repo *db.CallbackRepository) *Processor {
	return &Processor{repo: repo}
}

func (p *Processor) Process(ctx context.Context, event message.PaymentEvent) error {
	log.Printf("Processing event: %+v", event)

	callbackPayload := payload.Callback{
		ID:     event.Payload.ID,
		Status: event.Payload.Status,
	}

	payloadBytes, err := json.Marshal(callbackPayload)
	if err != nil {
		log.Printf("Error marshalling payload: %v", err)
		return err
	}

	callbackMessage := message.Callback{
		ID:        event.ID,
		PaymentID: event.Payload.ID,
		Url:       event.Payload.CallbackUrl,
		Payload:   string(payloadBytes),
	}

	messageBytes, err := json.Marshal(callbackMessage)
	if err != nil {
		log.Printf("Error marshalling message: %v", err)
		return err
	}

	entity := &db.CallbackEntity{
		ID:        event.ID,
		PaymentID: event.Payload.ID,
		Payload:   string(messageBytes),
		CreatedAt: time.Now(),
	}

	_, err = p.repo.Create(ctx, entity)
	if err != nil {
		log.Printf("Error creating entity: %v", err)
		return err
	}

	log.Printf("Successfully processed event with ID: %s", event.ID)
	return nil
}
