-- +goose Up
CREATE INDEX idx_callback_message_scheduled_at
    ON callback_message (scheduled_at)
    WHERE scheduled_at IS NOT NULL;