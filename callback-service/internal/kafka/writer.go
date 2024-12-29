package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

func NewWriter(kafkaURL, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(kafkaURL),
		Topic:                  topic,
		Balancer:               &kafka.ReferenceHash{},
		BatchSize:              100,
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           100 * time.Millisecond,
		Async:                  false,
		AllowAutoTopicCreation: false,
	}
}
