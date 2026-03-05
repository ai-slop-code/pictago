package requestctx

import (
	"context"
	"net/http"
	"testing"
)

func TestWithClientIP_ClientIP(t *testing.T) {
	ctx := context.Background()
	if got := ClientIP(ctx); got != "" {
		t.Errorf("ClientIP(empty ctx) = %q, want \"\"", got)
	}
	ctx = WithClientIP(ctx, "192.168.1.1")
	if got := ClientIP(ctx); got != "192.168.1.1" {
		t.Errorf("ClientIP(ctx with IP) = %q, want 192.168.1.1", got)
	}
}

func TestClientIPFromRequest(t *testing.T) {
	tests := []struct {
		name   string
		header string
		remote string
		wantIP string
	}{
		{"no header uses RemoteAddr", "", "1.2.3.4:5678", "1.2.3.4:5678"},
		{"X-Forwarded-For single", "10.0.0.1", "", "10.0.0.1"},
		{"X-Forwarded-For first of list", "10.0.0.1, 10.0.0.2, 10.0.0.3", "", "10.0.0.1"},
		{"X-Forwarded-For trimmed", "  10.0.0.1  ", "", "10.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/", nil)
			if tt.header != "" {
				r.Header.Set("X-Forwarded-For", tt.header)
			}
			r.RemoteAddr = tt.remote
			got := ClientIPFromRequest(r)
			if got != tt.wantIP {
				t.Errorf("ClientIPFromRequest() = %q, want %q", got, tt.wantIP)
			}
		})
	}
}

func TestWithRequestID_RequestID(t *testing.T) {
	ctx := context.Background()
	if got := RequestID(ctx); got != "" {
		t.Errorf("RequestID(empty) = %q", got)
	}
	ctx = WithRequestID(ctx, "req-abc")
	if got := RequestID(ctx); got != "req-abc" {
		t.Errorf("RequestID = %q", got)
	}
}

func TestRequestURL(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if got := RequestURL(nil); got != "" {
			t.Errorf("RequestURL(nil) = %q, want \"\"", got)
		}
	})
	t.Run("no scheme uses http and host", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "/api/files", nil)
		r.Host = "localhost:8080"
		got := RequestURL(r)
		if got != "http://localhost:8080/api/files" {
			t.Errorf("RequestURL() = %q, want http://localhost:8080/api/files", got)
		}
	})
	t.Run("with scheme preserved", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "https://example.com/path", nil)
		r.URL.Scheme = "https"
		r.URL.Host = "example.com"
		got := RequestURL(r)
		if got != "https://example.com/path" {
			t.Errorf("RequestURL() = %q", got)
		}
	})
}
