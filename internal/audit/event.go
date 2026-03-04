// Package audit writes application events in Elastic Common Schema (ECS) format.
package audit

import "time"

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
	Kind     string   `json:"kind,omitempty"`
	Category []string `json:"category,omitempty"`
	Action   string   `json:"action,omitempty"`
	Type     []string `json:"type,omitempty"`
	Outcome  string   `json:"outcome,omitempty"`
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

// EventOpts is used by BuildEvent to construct an Event from request context and action params.
type EventOpts struct {
	Action     string
	Category   []string
	Type       []string
	Outcome    string
	Message    string
	User       *UserFields
	File       *FileFields
	URL        string
	ClientIP   string
	StatusCode int
	Method     string
	Extra      map[string]interface{}
}

// BuildEvent builds an ECS Event from opts. Callers get ClientIP from request context (middleware).
func BuildEvent(opts EventOpts) Event {
	ev := Event{
		Event: EventFields{
			Category: opts.Category,
			Action:   opts.Action,
			Type:     opts.Type,
			Outcome:  opts.Outcome,
		},
		Message: opts.Message,
		User:    opts.User,
		File:    opts.File,
		Extra:   opts.Extra,
	}
	if opts.URL != "" {
		ev.URL = &URLFields{Full: opts.URL}
	}
	if opts.ClientIP != "" {
		ev.Client = &ClientFields{IP: opts.ClientIP}
	}
	if opts.Method != "" || opts.StatusCode != 0 {
		ev.HTTP = &HTTPFields{
			RequestMethod:      opts.Method,
			ResponseStatusCode: int64(opts.StatusCode),
		}
	}
	return ev
}
