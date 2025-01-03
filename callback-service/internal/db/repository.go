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

func (r *CallbackRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *CallbackRepository) Create(ctx context.Context, entity *CallbackMessageEntity) (*CallbackMessageEntity, error) {
	now := time.Now()

	query := `INSERT INTO callback_message (id, payment_id, url, payload, created_at, updated_at, scheduled_at, delivery_attempts, publish_attempts) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	err := r.pool.QueryRow(ctx, query, entity.ID, entity.PaymentID, entity.Url, entity.Payload, now, now, entity.ScheduledAt, entity.DeliveryAttempts, entity.PublishAttempts).Scan(&entity.ID)

	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *CallbackRepository) GetUnprocessedCallbacks(ctx context.Context, tx pgx.Tx, limit int) ([]*CallbackMessageEntity, error) {
	query := `SELECT id, payment_id, payload, url, delivery_attempts, publish_attempts
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
		if err := rows.Scan(&callback.ID, &callback.PaymentID, &callback.Payload, &callback.Url, &callback.DeliveryAttempts, &callback.PublishAttempts); err != nil {
			return nil, err
		}
		callbacks = append(callbacks, &callback)
	}
	return callbacks, nil
}

func (r *CallbackRepository) Update(ctx context.Context, tx pgx.Tx, entity *CallbackMessageEntity) error {
	query := `UPDATE callback_message 
	          SET payment_id = $1, url = $2, payload = $3, updated_at = $4, 
	              scheduled_at = $5, delivered_at = $6, delivery_attempts = $7, publish_attempts = $8, error = $9
	          WHERE id = $10`
	_, err := tx.Exec(ctx, query, entity.PaymentID, entity.Url, entity.Payload, time.Now(),
		entity.ScheduledAt, entity.DeliveredAt, entity.DeliveryAttempts, entity.PublishAttempts, entity.Error, entity.ID)
	return err
}

func (r *CallbackRepository) SelectForUpdateByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*CallbackMessageEntity, error) {
	query := `SELECT id, payment_id, payload, url, delivery_attempts, publish_attempts, scheduled_at, delivered_at, error, created_at, updated_at
	          FROM callback_message
	          WHERE id = $1
	          FOR UPDATE`
	row := tx.QueryRow(ctx, query, id)

	var entity CallbackMessageEntity
	err := row.Scan(&entity.ID, &entity.PaymentID, &entity.Payload, &entity.Url, &entity.DeliveryAttempts, &entity.PublishAttempts, &entity.ScheduledAt, &entity.DeliveredAt, &entity.Error, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *CallbackRepository) SelectByID(ctx context.Context, id uuid.UUID) (*CallbackMessageEntity, error) {
	query := `SELECT id, payment_id, payload, url, delivery_attempts, publish_attempts, scheduled_at, delivered_at, error, created_at, updated_at
	          FROM callback_message
	          WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)

	var entity CallbackMessageEntity
	err := row.Scan(&entity.ID, &entity.PaymentID, &entity.Payload, &entity.Url, &entity.DeliveryAttempts, &entity.PublishAttempts, &entity.ScheduledAt, &entity.DeliveredAt, &entity.Error, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}
