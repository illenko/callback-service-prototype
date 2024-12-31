package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"callback-service/internal/callback"
	"callback-service/internal/db"
	"callback-service/internal/event"
	"callback-service/internal/kafka"
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

	eventReader := kafka.NewReader("localhost:9092", "payment-events", "callback-service")
	defer eventReader.Close()

	go kafka.ReadPaymentEvents(eventReader, processor)

	callbackWriter := kafka.NewWriter("localhost:9092", "callback-messages")
	defer callbackWriter.Close()

	callbackProducer := callback.NewProducer(repo, callbackWriter)

	go callbackProducer.Start(context.Background())

	callbackSender := callback.NewSender()

	callbackProcessor := callback.NewCallbackProcessor(repo, callbackSender)

	go kafka.ReadCallbackMessages(kafka.NewReader("localhost:9092", "callback-messages", "callback-service"), callbackProcessor)

	log.Fatal(http.ListenAndServe(":8081", mux))
}
