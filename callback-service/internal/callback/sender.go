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
	"github.com/pkg/errors"
)

type Sender struct {
	client *http.Client
	logger *slog.Logger
}

func NewSender(cfg config.CallbackSender, logger *slog.Logger) *Sender {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	return &Sender{
		client: &http.Client{Timeout: timeout},
		logger: logger.With("component", "callback.sender"),
	}
}

func (s *Sender) Send(ctx context.Context, url, payload string) error {
	startTime := time.Now()

	s.logger.InfoContext(ctx, fmt.Sprintf("Sending callback to %s with payload %s", url, payload))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		s.logger.ErrorContext(ctx, fmt.Sprintf("Error creating request to %s", url), "error", err)
		return errors.Wrap(err, "creating http request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.ErrorContext(ctx, fmt.Sprintf("Error sending request to %s", url), "error", err)
		return errors.Wrap(err, "sending http request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.ErrorContext(ctx, fmt.Sprintf("Error reading response body from %s", url), "error", err)
		return errors.Wrap(err, "reading response body")
	}

	metrics.GetOrCreateHistogram(fmt.Sprintf(`callback_sender_milliseconds{url="%s",status="%d"}`, url, resp.StatusCode)).Update(float64(time.Since(startTime).Milliseconds()))

	s.logger.InfoContext(ctx, fmt.Sprintf("Received response from %s: %s", url, string(respBody)))

	if resp.StatusCode != http.StatusOK {
		s.logger.ErrorContext(ctx, fmt.Sprintf("Error response from %s: %s", url, resp.Status))
		return errors.Errorf("unexpected response status: %s", resp.Status)
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("Callback sent successfully to %s", url))
	return nil
}
