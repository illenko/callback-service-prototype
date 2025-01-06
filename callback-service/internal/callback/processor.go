package callback

import (
	"context"
	"log/slog"
	"time"

	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/logcontext"
	"callback-service/internal/message"
	"github.com/VictoriaMetrics/metrics"
)

const (
	defaultParallelism  = 1000
	defaultMaxAttempts  = 3
	defaultRetryDelayMs = 10_000
)

var (
	processErrorSendingCounter = metrics.GetOrCreateCounter(`callback_processor_total{result="error_sending"}`)
	processErrorTxCounter      = metrics.GetOrCreateCounter(`callback_processor_total{result="error_tx"}`)
	processErrorUpdateCounter  = metrics.GetOrCreateCounter(`callback_processor_total{result="error_update"}`)
	processSuccessCounter      = metrics.GetOrCreateCounter(`callback_processor_total{result="successfully_processed"}`)
	maxAttemptsCounter         = metrics.GetOrCreateCounter(`callback_processor_total{result="max_attempts"}`)
	rescheduledCounter         = metrics.GetOrCreateCounter(`callback_processor_total{result="rescheduled"}`)
	successCounter             = metrics.GetOrCreateCounter(`callback_processor_total{result="success"}`)
)

type Processor struct {
	repo        *db.CallbackRepository
	sender      *Sender
	sem         chan struct{}
	maxAttempts int
	retryDelay  time.Duration
	logger      *slog.Logger
}

func NewCallbackProcessor(repo *db.CallbackRepository, sender *Sender, logger *slog.Logger) *Processor {
	return &Processor{
		repo:        repo,
		sender:      sender,
		sem:         make(chan struct{}, config.GetInt("CALLBACK_PROCESSING_PARALLELISM", defaultParallelism)),
		maxAttempts: config.GetInt("MAX_DELIVERY_ATTEMPTS", defaultMaxAttempts),
		retryDelay:  time.Duration(config.GetInt("CALLBACK_RETRY_DELAY_MS", defaultRetryDelayMs)) * time.Millisecond,
		logger:      logger,
	}
}

func (p *Processor) Process(ctx context.Context, message message.Callback) error {
	ctx = logcontext.AppendCtx(ctx, slog.String("id", message.ID.String()))

	p.logger.InfoContext(ctx, "Processing callback message")

	p.sem <- struct{}{}
	go func() {
		defer func() { <-p.sem }()
		if err := p.processMessage(ctx, message); err != nil {
			p.logger.ErrorContext(ctx, "Failed to process message", "error", err)
		}
	}()

	return nil
}

func (p *Processor) processMessage(ctx context.Context, message message.Callback) error {
	callbackSendingErr := p.sender.Send(ctx, message.Url, message.Payload)
	if callbackSendingErr != nil {
		p.logger.ErrorContext(ctx, "Error sending callback", "error", callbackSendingErr)
		processErrorSendingCounter.Inc()
	}

	tx, err := p.repo.BeginTx(ctx)
	if err != nil {
		p.logger.ErrorContext(ctx, "Error starting transaction", "error", err)
		processErrorTxCounter.Inc()
		return err
	}
	defer tx.Rollback(ctx)

	entity, err := p.repo.SelectForUpdateByID(ctx, tx, message.ID)
	if err != nil {
		p.logger.ErrorContext(ctx, "Error selecting for update by ID", "error", err)
		processErrorTxCounter.Inc()
		return err
	}

	p.updateEntity(ctx, entity, callbackSendingErr)

	if err := p.repo.Update(ctx, tx, entity); err != nil {
		p.logger.ErrorContext(ctx, "Error updating callback", "error", err)
		processErrorUpdateCounter.Inc()
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.ErrorContext(ctx, "Error committing transaction", "error", err)
		processErrorTxCounter.Inc()
		return err
	}

	p.logger.InfoContext(ctx, "Transaction committed successfully")
	processSuccessCounter.Inc()
	return nil
}

func (p *Processor) updateEntity(ctx context.Context, entity *db.CallbackMessageEntity, callbackSendingErr error) {
	entity.DeliveryAttempts++

	if callbackSendingErr != nil {
		if entity.DeliveryAttempts >= p.maxAttempts {
			p.logger.WarnContext(ctx, "Max delivery attempts reached")
			entity.ScheduledAt = nil
			errorMsg := "Max delivery attempts reached. " + callbackSendingErr.Error()
			entity.Error = &errorMsg
			maxAttemptsCounter.Inc()
		} else {
			scheduledAt := time.Now().Add(time.Duration(entity.DeliveryAttempts) * p.retryDelay)
			errorMsg := callbackSendingErr.Error()
			entity.ScheduledAt = &scheduledAt
			entity.Error = &errorMsg
			entity.PublishAttempts = 0
			rescheduledCounter.Inc()
		}
	} else {
		p.logger.InfoContext(ctx, "Successfully processed and sent event")
		now := time.Now()
		entity.DeliveredAt = &now
		entity.ScheduledAt = nil
		entity.Error = nil
		successCounter.Inc()
	}
}
