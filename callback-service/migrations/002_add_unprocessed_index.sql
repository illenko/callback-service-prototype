-- +goose Up
CREATE INDEX IF NOT EXISTS idx_callback_messages_unprocessed
    ON callback_message (created_at, processed_at)
    INCLUDE (id, payment_id)
    WHERE processed_at IS NULL;