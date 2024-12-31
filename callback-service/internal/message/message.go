package message

import (
	"callback-service/internal/payload"
	"github.com/google/uuid"
)

type PaymentEvent struct {
	ID      uuid.UUID       `json:"id"`
	Event   string          `json:"event"`
	Payload payload.Payment `json:"payload"`
}

type Callback struct {
	ID        uuid.UUID `json:"id"`
	PaymentID uuid.UUID `json:"paymentId"`
	Url       string    `json:"url"`
	Payload   string    `json:"payload"`
	Attempts  int       `json:"attempts"`
}
