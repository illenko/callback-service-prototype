package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type CallbackRepository struct {
	pool *pgxpool.Pool
}

func NewCallbackRepository(pool *pgxpool.Pool) *CallbackRepository {
	return &CallbackRepository{pool: pool}
}

func (r *CallbackRepository) Create(ctx context.Context, entity *CallbackMessageEntity) (*CallbackMessageEntity, error) {
	query := `INSERT INTO callback_message (id, payment_id, url, payload, created_at, updated_at, scheduled_at, attempts) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err := r.pool.QueryRow(ctx, query, entity.ID, entity.PaymentID, entity.Url, entity.Payload,
		entity.CreatedAt, entity.UpdatedAt, entity.ScheduledAt, entity.Attempts).Scan(&entity.ID)

	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *CallbackRepository) ClearScheduledAt(ctx context.Context, tx pgx.Tx, entity *CallbackMessageEntity) error {
	query := `UPDATE callback_message 
	          SET updated_at = $2, scheduled_at = $3
	          WHERE id = $1`
	_, err := tx.Exec(ctx, query, entity.ID, time.Now(), entity.ScheduledAt)
	return err
}

func (r *CallbackRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *CallbackRepository) GetUnprocessedCallbacks(ctx context.Context, tx pgx.Tx, limit int) ([]*CallbackMessageEntity, error) {
	query := `SELECT id, payment_id, payload, url, attempts
	          FROM callback_message 
	          WHERE scheduled_at IS NOT NULL AND scheduled_at <= NOW()
	          LIMIT $1 
	          FOR UPDATE SKIP LOCKED`
	rows, err := tx.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var callbacks []*CallbackMessageEntity
	for rows.Next() {
		var callback CallbackMessageEntity
		if err := rows.Scan(&callback.ID, &callback.PaymentID, &callback.Payload, &callback.Url, &callback.Attempts); err != nil {
			return nil, err
		}
		callbacks = append(callbacks, &callback)
	}
	return callbacks, nil
}

func (r *CallbackRepository) SelectForUpdateByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*CallbackMessageEntity, error) {
	query := `SELECT id, payment_id, payload, url, attempts, scheduled_at, delivered_at, error
	          FROM callback_message
	          WHERE id = $1
	          FOR UPDATE`
	row := tx.QueryRow(ctx, query, id)

	var entity CallbackMessageEntity
	err := row.Scan(&entity.ID, &entity.PaymentID, &entity.Payload, &entity.Url, &entity.Attempts, &entity.ScheduledAt, &entity.DeliveredAt, &entity.Error)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *CallbackRepository) UpdateScheduledAtAndAttemptsByID(ctx context.Context, tx pgx.Tx, id uuid.UUID, scheduledAt time.Time, attempts int, errorMsg string) error {
	query := `UPDATE callback_message
	          SET scheduled_at = $2, attempts = $3, updated_at = $4, error = $5
	          WHERE id = $1`
	_, err := tx.Exec(ctx, query, id, scheduledAt, attempts, time.Now(), errorMsg)
	return err
}

func (r *CallbackRepository) UpdateAttemptsAndDeliveredAtByID(ctx context.Context, tx pgx.Tx, id uuid.UUID, attempts int, deliveredAt time.Time) error {
	query := `UPDATE callback_message
	          SET attempts = $2, delivered_at = $3, updated_at = $4
	          WHERE id = $1`
	_, err := tx.Exec(ctx, query, id, attempts, deliveredAt, time.Now())
	return err
}
