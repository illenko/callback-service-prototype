package metrics

import (
	"log"
	"time"

	"callback-service/internal/config"
	"github.com/VictoriaMetrics/metrics"
)

func Setup(cfg config.Metrics) {
	if cfg.URL == "" {
		return
	}

	err := metrics.InitPush(cfg.URL, time.Duration(cfg.IntervalMs)*time.Millisecond, cfg.CommonLabels, true)
	if err != nil {
		log.Printf("Error initializing metrics push: %v", err)
	}

}
