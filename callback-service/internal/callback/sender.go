package callback

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"callback-service/internal/config"
)

const (
	defaultTimeoutMs = 10_000
)

type Sender struct {
	client *http.Client
}

func NewSender() *Sender {
	timeout := time.Duration(config.GetInt("CALLBACK_TIMEOUT_MS", defaultTimeoutMs)) * time.Millisecond
	return &Sender{
		client: &http.Client{Timeout: timeout},
	}
}

func (s *Sender) Send(ctx context.Context, url, payload string) error {
	log.Printf("Sending callback to URL: %s", url)
	log.Printf("Request payload: %s", payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Error sending callback: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return err
	}

	log.Printf("Response status: %s", resp.Status)
	log.Printf("Response body: %s", string(respBody))

	if resp.StatusCode >= 400 {
		log.Printf("Received error response: %s", resp.Status)
		return fmt.Errorf("error response: %s", resp.Status)
	}

	log.Printf("Successfully sent callback to URL: %s", url)
	return nil
}
