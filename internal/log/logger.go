package log

import (
	"context"
	"log/slog"
	"os"
)

func New(opts ...Option) *slog.Logger {
	config := &Config{}

	for i := range opts {
		opts[i].apply(config)
	}

	if config.writer == nil {
		config.writer = os.Stderr
	}

	var (
		logLevel slog.Level // default is LevelInfo as a zero (int) value
		err      error
	)

	if config.level != "" {
		err = logLevel.UnmarshalText([]byte(config.level))
	}

	logger := slog.New(
		NewSpanContextHandler(
			slog.NewJSONHandler(config.writer, &slog.HandlerOptions{
				AddSource: config.addSource,
				Level:     logLevel,
			}),
			true,
		),
	)

	if err != nil {
		logger.WarnContext(context.Background(), "invalid log level string",
			slog.String("input_level", config.level),
			slog.String("error", err.Error()),
		)
	}

	return logger
}

// NewDefaultLogger initializes a logger with default settings.
// It uses the LOG_LEVEL environment variable to set the log level
// and includes source information in the logs.
func NewDefaultLogger() *slog.Logger {
	return New(
		WithLevel(os.Getenv("LOG_LEVEL")),
		WithSource(),
	)
}

func NoOp() *slog.Logger {
	return slog.New(noOpHandler{})
}

type noOpHandler struct{}

func (noOpHandler) Enabled(context.Context, slog.Level) bool {
	return false
}

func (noOpHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (h noOpHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h noOpHandler) WithGroup(string) slog.Handler {
	return h
}
