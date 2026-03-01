// Package audit writes application events in Elastic Common Schema (ECS) format
// to a configurable file path (e.g. audit.json).
package audit

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Event represents an ECS-formatted audit event.
// See https://www.elastic.co/guide/en/ecs/current/index.html
type Event struct {
	Timestamp time.Time              `json:"@timestamp"`
	Event     EventFields            `json:"event"`
	User      *UserFields            `json:"user,omitempty"`
	HTTP      *HTTPFields            `json:"http,omitempty"`
	File      *FileFields            `json:"file,omitempty"`
	URL       *URLFields             `json:"url,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Log       *LogFields             `json:"log,omitempty"`
	Client    *ClientFields          `json:"client,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

// UserFields can nest user.target for IAM events (e.g. user created by admin).
type UserFields struct {
	Name   string      `json:"name,omitempty"`
	ID     string      `json:"id,omitempty"`
	Target *UserFields `json:"target,omitempty"`
}

type EventFields struct {
	Kind     string   `json:"kind,omitempty"`     // event
	Category []string `json:"category,omitempty"`
	Action   string   `json:"action,omitempty"`
	Type     []string `json:"type,omitempty"`
	Outcome  string   `json:"outcome,omitempty"` // success, failure, unknown
	Dataset  string   `json:"dataset,omitempty"`
	ID       string   `json:"id,omitempty"`
	Module   string   `json:"module,omitempty"`
}


type HTTPFields struct {
	RequestMethod      string `json:"request.method,omitempty"`
	ResponseStatusCode int64  `json:"response.status_code,omitempty"`
}

type FileFields struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
	Size int64  `json:"size,omitempty"`
}

type URLFields struct {
	Full string `json:"full,omitempty"`
}

type LogFields struct {
	Logger string `json:"logger,omitempty"`
}

type ClientFields struct {
	IP string `json:"ip,omitempty"`
}

// Logger writes ECS audit events to a file (one JSON object per line).
type Logger struct {
	mu     sync.Mutex
	path   string
	writer io.Writer
	module string
}

// New creates a new audit logger. Path is the file path to write to (e.g. audit.json).
// The file is opened for append; parent directories are created if needed.
// If path is empty, Log methods no-op.
func New(path string) (*Logger, error) {
	if path == "" {
		return &Logger{}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{path: path, writer: f, module: "pictago"}, nil
}

// Close closes the underlying file. Safe to call if path was empty.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if c, ok := l.writer.(io.Closer); ok && c != nil {
		return c.Close()
	}
	return nil
}

func randomEventID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// Log writes a single ECS event as one JSON line. Thread-safe.
// If the logger was created with an empty path, Log is a no-op.
func (l *Logger) Log(ev Event) {
	l.mu.Lock()
	w := l.writer
	l.mu.Unlock()
	if w == nil {
		return
	}
	if ev.Event.ID == "" {
		ev.Event.ID = randomEventID()
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Event.Kind == "" {
		ev.Event.Kind = "event"
	}
	if ev.Event.Module == "" {
		ev.Event.Module = l.module
	}
	if ev.Event.Dataset == "" {
		ev.Event.Dataset = "pictago.audit"
	}
	payload := map[string]interface{}{
		"@timestamp": ev.Timestamp.Format(time.RFC3339Nano),
		"event":     ev.Event,
		"message":   ev.Message,
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
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.writer != nil {
		l.writer.Write(raw)
		l.writer.Write([]byte("\n"))
	}
}

// clientIP returns the client IP from the request (X-Forwarded-For or RemoteAddr).
func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		if idx := strings.Index(x, ","); idx >= 0 {
			return strings.TrimSpace(x[:idx])
		}
		return strings.TrimSpace(x)
	}
	return r.RemoteAddr
}
