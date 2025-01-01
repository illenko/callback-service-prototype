package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"callback-service/internal/callback"
	"callback-service/internal/db"
	"callback-service/internal/event"
	"callback-service/internal/kafka"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	time.Local = time.UTC

	mux := http.NewServeMux()
	mux.HandleFunc("GET /liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	conn, err := db.GetConn()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	if err := db.RunMigrations(conn); err != nil {
		log.Fatal(err)
	}

	dbpool, err := db.GetPool()
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()

	repo := db.NewCallbackRepository(dbpool)

	processor := event.NewProcessor(repo)

	kafkaURL := getRequiredVar("KAFKA_URL")
	paymentEventsTopic := getRequiredVar("PAYMENT_EVENTS_TOPIC")
	callbackMessagesTopic := getRequiredVar("CALLBACK_MESSAGES_TOPIC")
	callbackServiceGroupID := getRequiredVar("CALLBACK_SERVICE_GROUP_ID")
	port := getRequiredVar("PORT")

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

	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func getRequiredVar(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required but not set", key)
	}
	return value
}
