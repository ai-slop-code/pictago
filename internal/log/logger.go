package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
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

type Logger struct {
	*slog.Logger
}

func (log *Logger) Log(ctx context.Context, level logging.Level, msg string, fields ...any) {
	switch level {
	case logging.LevelDebug:
		log.Logger.DebugContext(ctx, msg, fields...)
	case logging.LevelInfo:
		log.Logger.InfoContext(ctx, msg, fields...)
	case logging.LevelWarn:
		log.Logger.WarnContext(ctx, msg, fields...)
	case logging.LevelError:
		log.Logger.ErrorContext(ctx, msg, fields...)
	default:
		log.Logger.InfoContext(ctx, msg, fields...)
	}
}

var _ logging.Logger = &Logger{}

// NewDefaultLogger initializes a logger with default settings.
// It uses the LOG_LEVEL environment variable to set the log level
// and includes source information in the logs.
func NewDefaultLogger() Logger {
	logger := Logger{}
	logger.Logger = New(
		WithLevel(os.Getenv("LOG_LEVEL")),
		WithSource(),
	)
	return logger
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
