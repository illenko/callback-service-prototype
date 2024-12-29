package model

import (
	"time"

	"github.com/google/uuid"
)

type Payload struct {
	ID          uuid.UUID `json:"id"`
	Amount      int       `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CallbackUrl string    `json:"callbackUrl"`
}

type PaymentEvent struct {
	ID      uuid.UUID `json:"id"`
	Event   string    `json:"event"`
	Payload Payload   `json:"payload"`
}

type CallbackPayload struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

type CallbackEntity struct {
	ID          uuid.UUID `json:"id"`
	PaymentID   uuid.UUID `json:"paymentId"`
	Payload     string    `json:"payload"`
	CreatedAt   time.Time `json:"createdAt"`
	ProcessedAt time.Time `json:"processedAt,omitempty"`
	Error       bool      `json:"error"`
}
