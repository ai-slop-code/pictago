package log

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("Should create logger with valid level", func(t *testing.T) {
		logger := New(WithLevel("debug"))
		assert.NotNil(t, logger)
	})

	t.Run("Should fallback to default level on invalid input", func(t *testing.T) {
		logger := New(WithLevel("notalevel"))
		assert.NotNil(t, logger)
	})

	t.Run("Should use custom writer", func(t *testing.T) {
		logger := New(WithWriter(os.Stdout))
		assert.NotNil(t, logger)
	})
}

func TestNewDefaultLogger(t *testing.T) {
	t.Run("Should create default logger", func(t *testing.T) {
		logger := NewDefaultLogger()
		assert.NotNil(t, logger)
	})
}

func TestNoOp(t *testing.T) {
	t.Run("Should return a no-op logger", func(t *testing.T) {
		logger := NoOp()
		assert.NotNil(t, logger)
	})
}

func TestNoOpHandler(t *testing.T) {
	handler := noOpHandler{}

	t.Run("Enabled should always return false", func(t *testing.T) {
		assert.False(t, handler.Enabled(t.Context(), slog.LevelDebug))
	})

	t.Run("Handle should always return nil", func(t *testing.T) {
		assert.NoError(t, handler.Handle(t.Context(), slog.Record{}))
	})

	t.Run("WithAttrs should return itself", func(t *testing.T) {
		assert.Equal(t, handler, handler.WithAttrs([]slog.Attr{slog.String("foo", "bar")}))
	})

	t.Run("WithGroup should return itself", func(t *testing.T) {
		assert.Equal(t, handler, handler.WithGroup("group"))
	})
}
