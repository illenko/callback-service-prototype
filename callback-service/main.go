package main

import (
	"callback-service/internal/logcontext"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"callback-service/internal/callback"
	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/event"
	"callback-service/internal/kafka"
	"github.com/VictoriaMetrics/metrics"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	time.Local = time.UTC

	logger := slog.New(&logcontext.ContextHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})

	dbConnStr := db.GetConnStr()

	db.RunMigrations(dbConnStr, "migrations")

	dbpool, err := db.GetPool(dbConnStr)
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()

	repo := db.NewCallbackRepository(dbpool)

	processor := event.NewProcessor(repo, logger)

	kafkaURL := config.GetRequired("KAFKA_URL")
	paymentEventsTopic := config.GetRequired("PAYMENT_EVENTS_TOPIC")
	callbackMessagesTopic := config.GetRequired("CALLBACK_MESSAGES_TOPIC")
	callbackServiceGroupID := config.GetRequired("CALLBACK_SERVICE_GROUP_ID")
	serverPort := config.GetRequired("SERVER_PORT")

	eventReader := kafka.NewReader(kafkaURL, paymentEventsTopic, callbackServiceGroupID)
	defer eventReader.Close()

	kafka.ReadPaymentEvents(eventReader, processor)

	callbackWriter := kafka.NewWriter(kafkaURL, callbackMessagesTopic)
	defer callbackWriter.Close()

	callbackProducer := callback.NewProducer(repo, callbackWriter, logger)
	callbackProducer.Start(context.Background())

	callbackSender := callback.NewSender(logger)
	callbackProcessor := callback.NewCallbackProcessor(repo, callbackSender, logger)

	kafka.ReadCallbackMessages(kafka.NewReader(kafkaURL, callbackMessagesTopic, callbackServiceGroupID), callbackProcessor)

	metricsUrl := config.GetRequired("VICTORIAMETRICS_PUSH_URL")
	metricsInterval := time.Duration(config.GetInt("VICTORIAMETRICS_PUSH_INTERVAL_MS", 10_000)) * time.Millisecond
	metricsLabels := config.GetRequired("VICTORIA_METRICS_COMMON_LABELS")

	metrics.InitPush(metricsUrl, metricsInterval, metricsLabels, true)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /liveness", func(w http.ResponseWriter, r *http.Request) {
		metrics.GetOrCreateCounter("liveliness_check").Inc()
		w.WriteHeader(http.StatusOK)
	})
	log.Fatal(http.ListenAndServe(":"+serverPort, mux))
}
