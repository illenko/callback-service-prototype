package main

import (
	"log"
	"net/http"

	"callback-service/internal/db"
	"callback-service/internal/kafka"
	"callback-service/internal/service"
)

func main() {

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

	processor := service.NewPaymentEventProcessor(repo)

	eventReader := kafka.NewReader("localhost:9092", "payment-events", "callback-service")
	defer eventReader.Close()

	go kafka.ReadPaymentEvents(eventReader, processor)

	callbackWriter := kafka.NewWriter("localhost:9092", "callback-messages")
	defer callbackWriter.Close()

	go kafka.ReadCallbackMessages(kafka.NewReader("localhost:9092", "callback-messages", "callback-service"))

	log.Fatal(http.ListenAndServe(":8080", mux))

}
