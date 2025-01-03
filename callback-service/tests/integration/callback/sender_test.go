package callback

import (
	"context"
	"testing"
	"time"

	"callback-service/internal/callback"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

func TestSender_Send(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   func()
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name: "Success",
			mockResponse: func() {
				gock.New("http://example.com").
					Post("/callback").
					Reply(200).
					JSON(map[string]string{"status": "ok"})
			},
			expectedError: false,
		},
		{
			name: "Error",
			mockResponse: func() {
				gock.New("http://example.com").
					Post("/callback").
					Reply(500).
					JSON(map[string]string{"error": "internal server error"})
			},
			expectedError: true,
		},
		{
			name: "Timeout",
			mockResponse: func() {
				gock.New("http://example.com").
					Post("/callback").
					Reply(200).
					Delay(15 * time.Second)
			},
			expectedError:  true,
			expectedErrMsg: "Client.Timeout exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer gock.Off()
			tt.mockResponse()

			sender := callback.NewSender()
			ctx := context.Background()
			url := "http://example.com/callback"
			payload := `{"data":"test"}`

			err := sender.Send(ctx, url, payload)
			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, gock.IsDone())
		})
	}
}
