package kafka

import (
	"time"

	"callback-service/internal/config"
	"github.com/segmentio/kafka-go"
)

const (
	DefaultBatchSize    = 100
	DefaultBatchTimeout = 100
)

func NewWriter(kafkaURL, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(kafkaURL),
		Topic:                  topic,
		Balancer:               &kafka.ReferenceHash{},
		BatchSize:              config.GetInt("KAFKA_WRITER_BATCH_SIZE", DefaultBatchSize),
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           time.Duration(config.GetInt("KAFKA_WRITER_BATCH_TIMEOUT", DefaultBatchTimeout)) * time.Millisecond,
		Async:                  false,
		AllowAutoTopicCreation: false,
	}
}
