package db

import (
	"time"

	"github.com/google/uuid"
)

type CallbackMessageEntity struct {
	ID              uuid.UUID
	PaymentID       uuid.UUID
	Url             string
	Payload         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ScheduledAt     *time.Time
	PublishedAt     *time.Time
	DeliveredAt     *time.Time
	PublishAttempts int
	Attempts        int
	Error           *string
}
