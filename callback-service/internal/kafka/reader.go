package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"callback-service/internal/model"
	"callback-service/internal/service"
	"github.com/segmentio/kafka-go"
)

func NewReader(kafkaURL, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(kafkaURL, ","),
		GroupID: groupID,
		Topic:   topic,
	})
}

func ReadPaymentEvents(reader *kafka.Reader, processor *service.PaymentEventProcessor) {
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("message at topic:%v partition:%v offset:%v	%s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

		var event model.PaymentEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		_, err = processor.Process(context.Background(), event)
		if err != nil {
			log.Printf("Error processing event: %v", err)
			return
		}
	}
}

func ReadCallbackMessages(reader *kafka.Reader) {
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("message at topic:%v partition:%v offset:%v	%s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
	}
}
