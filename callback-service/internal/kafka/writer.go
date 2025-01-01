package kafka

import (
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

func NewWriter(kafkaURL, topic string) *kafka.Writer {
	batchSizeStr := os.Getenv("KAFKA_WRITER_BATCH_SIZE")
	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil || batchSizeStr == "" {
		batchSize = 100
	}

	batchTimeoutStr := os.Getenv("KAFKA_WRITER_BATCH_TIMEOUT")
	batchTimeout, err := strconv.Atoi(batchTimeoutStr)
	if err != nil || batchTimeoutStr == "" {
		batchTimeout = 100
	}

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
