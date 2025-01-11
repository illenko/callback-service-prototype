package config

import (
	"log"

	"github.com/spf13/viper"
)

type Database struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	SSLMode  string `mapstructure:"ssl-mode"`
}

type KafkaWriter struct {
	BatchSize      int `mapstructure:"batch-size"`
	BatchTimeoutMs int `mapstructure:"batch-timeout-ms"`
}

type KafkaBroker struct {
	URL string `mapstructure:"url"`
}

type KafkaTopic struct {
	PaymentEvents    string `mapstructure:"payment-events"`
	CallbackMessages string `mapstructure:"callback-messages"`
}

type KafkaReader struct {
	GroupID string `mapstructure:"group-id"`
}

type Kafka struct {
	Writer KafkaWriter `mapstructure:"writer"`
	Broker KafkaBroker `mapstructure:"broker"`
	Topic  KafkaTopic  `mapstructure:"topic"`
	Reader KafkaReader `mapstructure:"reader"`
}

type CallbackProcessor struct {
	Parallelism         int `mapstructure:"parallelism"`
	RescheduleDelayMs   int `mapstructure:"reschedule-delay-ms"`
	MaxDeliveryAttempts int `mapstructure:"max-delivery-attempts"`
}

type CallbackProducer struct {
	PollingIntervalMs  int `mapstructure:"polling-interval-ms"`
	FetchSize          int `mapstructure:"fetch-size"`
	RescheduleDelayMs  int `mapstructure:"reschedule-delay-ms"`
	MaxPublishAttempts int `mapstructure:"max-publish-attempts"`
}

type CallbackSender struct {
	TimeoutMs int `mapstructure:"timeout-ms"`
}

type Callback struct {
	Processor CallbackProcessor `mapstructure:"processor"`
	Producer  CallbackProducer  `mapstructure:"producer"`
	Sender    CallbackSender    `mapstructure:"sender"`
}

type Server struct {
	Port string `mapstructure:"port"`
}

type Metrics struct {
	URL          string `mapstructure:"url"`
	IntervalMs   int    `mapstructure:"interval-ms"`
	CommonLabels string `mapstructure:"common-labels"`
}

type Logs struct {
	URL string `mapstructure:"url"`
}

type Config struct {
	Database Database `mapstructure:"database"`
	Kafka    Kafka    `mapstructure:"kafka"`
	Callback Callback `mapstructure:"callback"`
	Server   Server   `mapstructure:"server"`
	Metrics  Metrics  `mapstructure:"metrics"`
	Logs     Logs     `mapstructure:"logs"`
}

func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func MustLoadConfig(path string) *Config {
	config, err := LoadConfig(path)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return config
}
