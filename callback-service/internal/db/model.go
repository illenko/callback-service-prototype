package db

import (
	"time"

	"github.com/google/uuid"
)

type CallbackMessageEntity struct {
	ID          uuid.UUID `json:"id"`
	PaymentID   uuid.UUID `json:"paymentId"`
	Payload     string    `json:"payload"`
	CreatedAt   time.Time `json:"createdAt"`
	ProcessedAt time.Time `json:"processedAt,omitempty"`
	Error       bool      `json:"error"`
}
