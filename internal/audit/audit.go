package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const (
	module  = "pictago"
	dataset = "pictago.audit"
)

// Logger writes ECS audit events. Implementations must be safe for concurrent use.
type Logger interface {
	Log(Event)
	Close() error
}

var _ Logger = (*logger)(nil)

type logger struct {
	log *logrus.Logger
}

// ecsFormatter outputs a single JSON object per line (ECS-style) from the entry's Data.
type ecsFormatter struct{}

func (ecsFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := entry.Data
	if data == nil {
		data = make(map[string]interface{})
	}
	if _, ok := data["@timestamp"]; !ok {
		data["@timestamp"] = entry.Time.UTC().Format(time.RFC3339Nano)
	}
	if entry.Message != "" {
		data["message"] = entry.Message
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// New creates a new audit logger using logrus, writing ECS JSON lines to path.
// If path is empty, Log is a no-op and Close is safe.
func New(path string) (Logger, error) {
	l := logrus.New()
	l.SetFormatter(ecsFormatter{})
	l.SetLevel(logrus.InfoLevel)
	l.SetReportCaller(false)

	if path == "" {
		l.SetOutput(io.Discard)
		return &logger{log: l}, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	l.SetOutput(f)
	return &logger{log: l}, nil
}

func (l *logger) Close() error {
	if l.log == nil {
		return nil
	}
	if w, ok := l.log.Out.(io.Closer); ok && w != nil {
		return w.Close()
	}
	return nil
}

// Log writes a single ECS event as one JSON line via logrus.
func (l *logger) Log(ev Event) {
	if l.log == nil || l.log.Out == io.Discard {
		return
	}
	if ev.Event.ID == "" {
		ev.Event.ID = uuid.New().String()
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Event.Kind == "" {
		ev.Event.Kind = "event"
	}
	if ev.Event.Module == "" {
		ev.Event.Module = module
	}
	if ev.Event.Dataset == "" {
		ev.Event.Dataset = dataset
	}

	payload := map[string]interface{}{
		"@timestamp": ev.Timestamp.Format(time.RFC3339Nano),
		"event":      ev.Event,
		"message":    ev.Message,
	}
	if ev.User != nil {
		payload["user"] = ev.User
	}
	if ev.HTTP != nil {
		payload["http"] = ev.HTTP
	}
	if ev.File != nil {
		payload["file"] = ev.File
	}
	if ev.URL != nil {
		payload["url"] = ev.URL
	}
	if ev.Log != nil {
		payload["log"] = ev.Log
	}
	if ev.Client != nil {
		payload["client"] = ev.Client
	}
	for k, v := range ev.Extra {
		payload[k] = v
	}

	l.log.WithFields(payload).Info("")
}

// WrapFilesHandler wraps the /files/ handler to log each GET/HEAD as a file-access event.
func WrapFilesHandler(logger Logger, next http.Handler, clientIP func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != "HEAD" {
			next.ServeHTTP(w, r)
			return
		}
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "files.public.access")
		defer span.End()
		r = r.WithContext(ctx)
		rw := &responseWriterWithStatus{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		outcome := "success"
		if rw.status >= 400 {
			outcome = "failure"
			span.SetStatus(codes.Error, "file access failed")
		}
		path := r.URL.Path
		if path == "" {
			path = "/"
		}
		urlStr := ""
		if r.URL != nil {
			urlStr = r.URL.String()
			if r.URL.Scheme == "" {
				urlStr = "http://" + r.Host + urlStr
			}
		}
		ev := BuildEvent(EventOpts{
			Action:     "file-access",
			Category:   []string{"file", "web"},
			Type:       []string{"access"},
			Outcome:    outcome,
			Message:    "public file access " + outcome + ": " + path,
			File:       &FileFields{Path: path},
			URL:        urlStr,
			ClientIP:   clientIP(r),
			StatusCode: rw.status,
			Method:     r.Method,
		})
		logger.Log(ev)
	})
}

type responseWriterWithStatus struct {
	http.ResponseWriter
	status int
}

func (w *responseWriterWithStatus) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// NoopLogger returns a Logger that discards all output (for when audit is disabled).
func NoopLogger() Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetFormatter(ecsFormatter{})
	return &logger{log: l}
}
