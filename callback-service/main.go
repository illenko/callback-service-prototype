package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"callback-service/internal/callback"
	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/event"
	"callback-service/internal/kafka"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	time.Local = time.UTC

	dbConnStr := db.GetConnStr()

	db.RunMigrations(dbConnStr, "migrations")

	dbpool, err := db.GetPool(dbConnStr)
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()

	repo := db.NewCallbackRepository(dbpool)

	processor := event.NewProcessor(repo)

	kafkaURL := config.GetRequired("KAFKA_URL")
	paymentEventsTopic := config.GetRequired("PAYMENT_EVENTS_TOPIC")
	callbackMessagesTopic := config.GetRequired("CALLBACK_MESSAGES_TOPIC")
	callbackServiceGroupID := config.GetRequired("CALLBACK_SERVICE_GROUP_ID")
	serverPort := config.GetRequired("SERVER_PORT")

	eventReader := kafka.NewReader(kafkaURL, paymentEventsTopic, callbackServiceGroupID)
	defer eventReader.Close()

	go kafka.ReadPaymentEvents(eventReader, processor)

	callbackWriter := kafka.NewWriter(kafkaURL, callbackMessagesTopic)
	defer callbackWriter.Close()

	callbackProducer := callback.NewProducer(repo, callbackWriter)
	go callbackProducer.Start(context.Background())

	callbackSender := callback.NewSender()
	callbackProcessor := callback.NewCallbackProcessor(repo, callbackSender)

	go kafka.ReadCallbackMessages(kafka.NewReader(kafkaURL, callbackMessagesTopic, callbackServiceGroupID), callbackProcessor)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	log.Fatal(http.ListenAndServe(":"+serverPort, mux))
}
