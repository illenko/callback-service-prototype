package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"callback-service/internal/callback"
	"callback-service/internal/event"
	"callback-service/internal/message"
	"github.com/segmentio/kafka-go"
)

func NewReader(kafkaURL, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(kafkaURL, ","),
		GroupID: groupID,
		Topic:   topic,
	})
}

func ReadPaymentEvents(reader *kafka.Reader, processor *event.Processor) {
	go func() {
		for {
			m, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Println(err)
				continue
			}
			fmt.Printf("message at topic:%v partition:%v offset:%v	%s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

			var e message.PaymentEvent
			if err := json.Unmarshal(m.Value, &e); err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}

			err = processor.Process(context.Background(), e)
			if err != nil {
				log.Printf("Error processing event: %v", err)
				return
			}
		}
	}()
}

func ReadCallbackMessages(reader *kafka.Reader, processor *callback.Processor) {
	go func() {
		for {
			m, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Println(err)
				continue
			}
			fmt.Printf("message at topic:%v partition:%v offset:%v	%s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

			var c message.Callback
			if err := json.Unmarshal(m.Value, &c); err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}

			err = processor.Process(context.Background(), c)
			if err != nil {
				log.Printf("Error processing callback message: %v", err)
				return
			}
		}
	}()
}
