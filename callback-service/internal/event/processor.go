package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"callback-service/internal/db"
	"callback-service/internal/logging"
	"callback-service/internal/message"
	"callback-service/internal/payload"
	"github.com/VictoriaMetrics/metrics"
	"github.com/google/uuid"
)

var (
	successCounter = metrics.GetOrCreateCounter(`event_processor_messages_total{result="success"}`)
	dropCounter    = metrics.GetOrCreateCounter(`event_processor_messages_total{result="drop"}`)
	failCounter    = metrics.GetOrCreateCounter(`event_processor_messages_total{result="fail"}`)
)

type Processor struct {
	repo   *db.CallbackRepository
	logger *slog.Logger
}

func NewProcessor(repo *db.CallbackRepository, logger *slog.Logger) *Processor {
	return &Processor{
		repo:   repo,
		logger: logger.With("component", "event.processor"),
	}
}

func (p *Processor) Process(ctx context.Context, event message.PaymentEvent) error {
	ctx = logging.AppendCtx(ctx, slog.String("correlationId", uuid.New().String()))
	ctx = logging.AppendCtx(ctx, slog.String("eventId", event.ID.String()))

	p.logger.InfoContext(ctx, fmt.Sprintf("Processing event %v", event))

	if event.Payload.Status != "successful" && event.Payload.Status != "failed" {
		dropCounter.Inc()

		p.logger.InfoContext(ctx, "Event dropped due to status")
		return nil
	}

	payloadBytes, err := json.Marshal(toPayload(event))
	if err != nil {
		failCounter.Inc()

		p.logger.ErrorContext(ctx, fmt.Sprintf("Error marshalling payload: %v", err))
		return err
	}

	_, err = p.repo.Create(ctx, toEntity(event, payloadBytes, time.Now()))
	if err != nil {
		failCounter.Inc()
		p.logger.ErrorContext(ctx, fmt.Sprintf("Error saving callback message: %v", err))
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
