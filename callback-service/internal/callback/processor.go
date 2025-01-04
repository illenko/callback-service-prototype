package callback

import (
	"context"
	"log"
	"time"

	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/message"
	"github.com/VictoriaMetrics/metrics"
)

const (
	defaultParallelism  = 1000
	defaultMaxAttempts  = 3
	defaultRetryDelayMs = 10_000
)

var (
	processErrorMetricSending = metrics.GetOrCreateCounter(`callback_processor_errors_total{stage="sending"}`)
	processErrorMetricTx      = metrics.GetOrCreateCounter(`callback_processor_errors_total{stage="transaction"}`)
	processErrorMetricUpdate  = metrics.GetOrCreateCounter(`callback_processor_errors_total{stage="update"}`)
	processSuccessMetric      = metrics.GetOrCreateCounter(`callback_processor_successful_total`)
	maxAttemptsMetric         = metrics.GetOrCreateCounter(`callback_processor_max_attempts_total`)
)

type Processor struct {
	repo        *db.CallbackRepository
	sender      *Sender
	sem         chan struct{}
	maxAttempts int
	retryDelay  time.Duration
}

func NewCallbackProcessor(repo *db.CallbackRepository, sender *Sender) *Processor {
	return &Processor{
		repo:        repo,
		sender:      sender,
		sem:         make(chan struct{}, config.GetInt("CALLBACK_PROCESSING_PARALLELISM", defaultParallelism)),
		maxAttempts: config.GetInt("MAX_DELIVERY_ATTEMPTS", defaultMaxAttempts),
		retryDelay:  time.Duration(config.GetInt("CALLBACK_RETRY_DELAY_MS", defaultRetryDelayMs)) * time.Millisecond,
	}
}

func (p *Processor) Process(ctx context.Context, message message.Callback) error {
	log.Printf("Processing message: %+v", message)

	p.sem <- struct{}{}
	go func() {
		defer func() { <-p.sem }()

		callbackSendingErr := p.sender.Send(ctx, message.Url, message.Payload)
		if callbackSendingErr != nil {
			log.Printf("Error sending callback: %v", callbackSendingErr)
			processErrorMetricSending.Inc()
		}

		tx, err := p.repo.BeginTx(ctx)
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			processErrorMetricTx.Inc()
			return
		}

		entity, err := p.repo.SelectForUpdateByID(ctx, tx, message.ID)
		if err != nil {
			log.Printf("Error selecting for update by ID: %v", err)
			tx.Rollback(ctx)
			processErrorMetricTx.Inc()
			return
		}

		entity.DeliveryAttempts++

		if callbackSendingErr != nil {
			// Check if we have reached the max number of attempts
			if entity.DeliveryAttempts >= p.maxAttempts {
				log.Printf("Max delivery attempts reached for callback ID: %s", message.ID)
				entity.ScheduledAt = nil
				errorMsg := "Max delivery attempts reached. " + callbackSendingErr.Error()
				entity.Error = &errorMsg
				maxAttemptsMetric.Inc()
			} else {
				scheduledAt := time.Now().Add(time.Duration(entity.DeliveryAttempts) * p.retryDelay)
				errorMsg := callbackSendingErr.Error()
				entity.ScheduledAt = &scheduledAt
				entity.Error = &errorMsg

				// Reset publish attempts
				entity.PublishAttempts = 0
			}

		} else {
			log.Printf("Successfully processed and sent event with ID: %s", message.ID)

			now := time.Now()

			entity.DeliveredAt = &now
			entity.ScheduledAt = nil
			entity.Error = nil
		}

		log.Printf("Updating callback with ID: %s", message.ID)

		err = p.repo.Update(ctx, tx, entity)
		if err != nil {
			log.Printf("Error updating callback: %v", err)
			tx.Rollback(ctx)
			processErrorMetricUpdate.Inc()
			return
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("Error committing transaction: %v", err)
			tx.Rollback(ctx)
			processErrorMetricTx.Inc()
		} else {
			log.Println("Transaction committed successfully")
			processSuccessMetric.Inc()
		}
	}()

	return nil
}
