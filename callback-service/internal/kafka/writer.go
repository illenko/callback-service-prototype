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
	batchSize := config.GetEnvInt("KAFKA_WRITER_BATCH_SIZE", DefaultBatchSize)
	batchTimeout := config.GetEnvInt("KAFKA_WRITER_BATCH_TIMEOUT", DefaultBatchTimeout)

	return &kafka.Writer{
		Addr:                   kafka.TCP(kafkaURL),
		Topic:                  topic,
		Balancer:               &kafka.ReferenceHash{},
		BatchSize:              batchSize,
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           time.Duration(batchTimeout) * time.Millisecond,
		Async:                  false,
		AllowAutoTopicCreation: false,
	}
}
