package callback

import (
	"context"
	"fmt"
	"log"
	"time"

	"callback-service/internal/config"
	"github.com/go-resty/resty/v2"
)

const (
	defaultTimeoutMs = 10_000
)

type Sender struct {
	client *resty.Client
}

func NewSender() *Sender {
	return &Sender{
		client: resty.New().
			SetTimeout(time.Duration(config.GetEnvInt("CALLBACK_TIMEOUT_MS", defaultTimeoutMs)) * time.Millisecond),
	}
}

func (s *Sender) Send(ctx context.Context, url, payload string) error {
	log.Printf("Sending callback to URL: %s", url)

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(url)

	if err != nil {
		log.Printf("Error sending callback: %v", err)
		return err
	}

	if resp.IsError() {
		log.Printf("Received error response: %s", resp.Status())
		return fmt.Errorf("error response: %s", resp.Status())
	}

	log.Printf("Successfully sent callback to URL: %s", url)
	return nil
}
