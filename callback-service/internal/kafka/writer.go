package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

func NewWriter(kafkaURL string, batchSize int, batchTimeoutMs int, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(kafkaURL),
		Topic:                  topic,
		Balancer:               &kafka.ReferenceHash{},
		BatchSize:              batchSize,
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           time.Duration(batchTimeoutMs) * time.Millisecond,
		Async:                  false,
		AllowAutoTopicCreation: false,
	}
}
