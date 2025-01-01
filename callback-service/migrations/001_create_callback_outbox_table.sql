-- +goose Up
CREATE TABLE callback_message
(
    id                UUID PRIMARY KEY,
    payment_id        UUID          NOT NULL,
    payload           JSONB         NOT NULL,
    url               VARCHAR(2048) NOT NULL,
    created_at        TIMESTAMP     NOT NULL,
    updated_at        TIMESTAMP     NOT NULL,
    scheduled_at      TIMESTAMP,
    delivered_at      TIMESTAMP,
    delivery_attempts INT           NOT NULL DEFAULT 0,
    error             TEXT          NULL
);