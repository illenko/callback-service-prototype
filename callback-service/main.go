package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	"callback-service/internal/callback"
	"callback-service/internal/config"
	"callback-service/internal/db"
	"callback-service/internal/event"
	"callback-service/internal/kafka"
	"callback-service/internal/logging"
	"callback-service/internal/metrics"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	time.Local = time.UTC

	cfg := config.MustLoadConfig("./")

	logger := logging.GetLogger(cfg.Logs)
	slog.SetDefault(logger)

	dbConnStr := db.GetConnStr(cfg.Database)

	db.RunMigrations(dbConnStr, "migrations")

	dbpool, err := db.GetPool(dbConnStr)
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()

	repo := db.NewCallbackRepository(dbpool)

	processor := event.NewProcessor(repo, logger)

	eventReader := kafka.NewReader(cfg.Kafka.Broker.URL, cfg.Kafka.Topic.PaymentEvents, cfg.Kafka.Reader.GroupID)
	defer eventReader.Close()

	kafka.ReadPaymentEvents(eventReader, processor, logger)

	callbackWriter := kafka.NewWriter(cfg.Kafka.Broker.URL, cfg.Kafka.Writer.BatchSize, cfg.Kafka.Writer.BatchTimeoutMs, cfg.Kafka.Topic.CallbackMessages)
	defer callbackWriter.Close()

	callbackSender := callback.NewSender(cfg.Callback.Sender, logger)
	callbackProcessor := callback.NewCallbackProcessor(repo, callbackSender, cfg.Callback.Processor, logger)

	kafka.ReadCallbackMessages(kafka.NewReader(cfg.Kafka.Broker.URL, cfg.Kafka.Topic.CallbackMessages, cfg.Kafka.Reader.GroupID), callbackProcessor, logger)

	metrics.Setup(cfg.Metrics)

	callbackProducer := callback.NewProducer(repo, callbackWriter, cfg.Callback.Producer, logger)
	callbackProducer.Start(context.Background())
}
