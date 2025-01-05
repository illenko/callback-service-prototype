package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"callback-service/internal/db"
	"callback-service/internal/message"
	"callback-service/internal/payload"
	"github.com/VictoriaMetrics/metrics"
)

var (
	successCounter = metrics.GetOrCreateCounter(`event_processor_messages_total{result="success"}`)
	failCounter    = metrics.GetOrCreateCounter(`event_processor_messages_total{result="fail"}`)
)

type Processor struct {
	repo   *db.CallbackRepository
	logger *slog.Logger
}

func NewProcessor(repo *db.CallbackRepository, logger *slog.Logger) *Processor {
	return &Processor{
		repo:   repo,
		logger: logger,
	}
}

func (p *Processor) Process(ctx context.Context, event message.PaymentEvent) error {
	ctx = context.WithValue(ctx, "id", event.ID)

	p.logger.InfoContext(ctx, "Processing event", "id", event.ID, "event", event)

	payloadBytes, err := json.Marshal(toPayload(event))
	if err != nil {
		failCounter.Inc()
		p.logger.ErrorContext(ctx, "Error marshalling payload", "error", err)
		return err
	}

	_, err = p.repo.Create(ctx, toEntity(event, payloadBytes, time.Now()))
	if err != nil {
		failCounter.Inc()
		p.logger.ErrorContext(ctx, "Error creating callback message", "error", err)
		return err
	}

	p.logger.InfoContext(ctx, "Event processed successfully")

	successCounter.Inc()

	return nil
}

func toEntity(event message.PaymentEvent, payloadBytes []byte, scheduledAt time.Time) *db.CallbackMessageEntity {
	return &db.CallbackMessageEntity{
		ID:          event.ID,
		PaymentID:   event.Payload.ID,
		Payload:     string(payloadBytes),
		Url:         event.Payload.CallbackUrl,
		ScheduledAt: &scheduledAt,
	}
}

func toPayload(event message.PaymentEvent) payload.Callback {
	return payload.Callback{
		ID:        event.ID,
		PaymentId: event.Payload.ID,
		Status:    event.Payload.Status,
	}
}
