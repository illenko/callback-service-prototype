package logging

import (
	"callback-service/internal/config"
	"context"
	"github.com/grafana/loki-client-go/loki"
	slogloki "github.com/samber/slog-loki/v3"
	"log/slog"
	"os"
)

func GetLogger(cfg config.Logs) *slog.Logger {
	if cfg.URL == "" {
		return localLogger()
	}

	return remoteLogger(cfg.URL)
}

func localLogger() *slog.Logger {
	return slog.New(&ContextHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})
}

func remoteLogger(url string) *slog.Logger {
	lokiConfig, _ := loki.NewDefaultConfig(url)
	client, _ := loki.New(lokiConfig)

	return slog.New(slogloki.Option{
		Level:  slog.LevelInfo,
		Client: client,
		AttrFromContext: []func(ctx context.Context) []slog.Attr{
			func(ctx context.Context) []slog.Attr {
				var attrs []slog.Attr
				if v, ok := ctx.Value(slogFields).([]slog.Attr); ok {
					attrs = append(attrs, v...)
				}
				return attrs
			},
		},
	}.NewLokiHandler()).With("service", "callback-service")
}
