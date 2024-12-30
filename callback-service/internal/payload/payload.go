package payload

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID          uuid.UUID `json:"id"`
	Amount      int       `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CallbackUrl string    `json:"callbackUrl"`
}

type Callback struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}
