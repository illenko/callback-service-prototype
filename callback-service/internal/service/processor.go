package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"callback-service/internal/db"
	"callback-service/internal/model"
)

type PaymentEventProcessor struct {
	repo *db.CallbackRepository
}

func NewPaymentEventProcessor(repo *db.CallbackRepository) *PaymentEventProcessor {
	return &PaymentEventProcessor{repo: repo}
}

func (p *PaymentEventProcessor) Process(ctx context.Context, event model.PaymentEvent) (*db.CallbackEntity, error) {
	log.Printf("Processing event: %+v", event)

	payload := model.CallbackPayload{
		ID:     event.Payload.ID,
		Status: event.Payload.Status,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload: %v", err)
		return nil, err
	}

	entity := &db.CallbackEntity{
		ID:        event.ID,
		PaymentID: event.Payload.ID,
		Payload:   string(payloadBytes),
		CreatedAt: time.Now(),
	}

	newEntity, err := p.repo.Create(ctx, entity)
	if err != nil {
		log.Printf("Error creating entity: %v", err)
		return nil, err
	}

	log.Printf("Successfully processed event with ID: %s", event.ID)
	return newEntity, nil
}
