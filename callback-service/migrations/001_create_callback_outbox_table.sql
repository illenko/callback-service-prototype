-- +goose Up
CREATE TABLE callback_message
(
    id           UUID PRIMARY KEY,
    payment_id   UUID      NOT NULL,
    payload      JSONB     NOT NULL,
    created_at   TIMESTAMP NOT NULL,
    processed_at TIMESTAMP,
    error        BOOLEAN DEFAULT FALSE
);