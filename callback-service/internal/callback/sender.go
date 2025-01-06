package callback

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"callback-service/internal/config"
	"github.com/VictoriaMetrics/metrics"
)

const (
	defaultTimeoutMs = 10_000
)

type Sender struct {
	client *http.Client
	logger *slog.Logger
}

func NewSender(logger *slog.Logger) *Sender {
	timeout := time.Duration(config.GetInt("CALLBACK_TIMEOUT_MS", defaultTimeoutMs)) * time.Millisecond
	return &Sender{
		client: &http.Client{Timeout: timeout},
		logger: logger,
	}
}

func (s *Sender) Send(ctx context.Context, url, payload string) error {
	startTime := time.Now()

	s.logger.InfoContext(ctx, "Sending callback", "url", url)
	s.logger.InfoContext(ctx, "Request payload", "payload", payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		s.logger.ErrorContext(ctx, "Error creating request", "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "Error sending callback", "error", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.ErrorContext(ctx, "Error reading response body", "error", err)
		return err
	}

	metrics.GetOrCreateHistogram(fmt.Sprintf(`callback_sender_milliseconds{url="%s",status="%d"}`, url, resp.StatusCode)).Update(float64(time.Since(startTime).Milliseconds()))

	s.logger.InfoContext(ctx, "Response received", "status", resp.Status, "body", string(respBody))

	if resp.StatusCode != http.StatusOK {
		s.logger.ErrorContext(ctx, "Received error response", "status", resp.Status)
		return fmt.Errorf("error response: %s", resp.Status)
	}

	s.logger.InfoContext(ctx, "Successfully sent callback", "url", url)
	return nil
}
