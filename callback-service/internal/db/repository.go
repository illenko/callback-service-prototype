package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CallbackEntity struct {
	ID          uuid.UUID `json:"id"`
	PaymentID   uuid.UUID `json:"paymentId"`
	Payload     string    `json:"payload"`
	CreatedAt   time.Time `json:"createdAt"`
	ProcessedAt time.Time `json:"processedAt,omitempty"`
	Error       bool      `json:"error"`
}

type CallbackRepository struct {
	pool *pgxpool.Pool
}

func NewCallbackRepository(pool *pgxpool.Pool) *CallbackRepository {
	return &CallbackRepository{pool: pool}
}

func (r *CallbackRepository) Create(ctx context.Context, entity *CallbackEntity) (*CallbackEntity, error) {
	query := `INSERT INTO callback_message (id, payment_id, payload, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err := r.pool.QueryRow(ctx, query, entity.ID, entity.PaymentID, entity.Payload, entity.CreatedAt).Scan(&entity.ID)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *CallbackRepository) GetByID(ctx context.Context, id uuid.UUID) (*CallbackEntity, error) {
	query := `SELECT id, payment_id, payload, created_at, processed_at, error FROM callback_message WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)

	var entity CallbackEntity
	err := row.Scan(&entity.ID, &entity.PaymentID, &entity.Payload, &entity.CreatedAt, &entity.ProcessedAt, &entity.Error)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *CallbackRepository) ResetProcessedAt(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE callback_message SET processed_at = NULL, error = false WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
