// Package requestctx provides request-scoped values (e.g. client IP) set by middleware.
package requestctx

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const keyClientIP contextKey = "client_ip"

// WithClientIP returns a context with the client IP set. Middleware should call this.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, keyClientIP, ip)
}

// ClientIP returns the client IP from the context, or empty string if not set.
func ClientIP(ctx context.Context) string {
	v, _ := ctx.Value(keyClientIP).(string)
	return v
}

// ClientIPFromRequest returns the client IP from the request (X-Forwarded-For or RemoteAddr).
// Middleware should call this and set the result in context via WithClientIP.
func ClientIPFromRequest(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		if idx := strings.Index(x, ","); idx >= 0 {
			return strings.TrimSpace(x[:idx])
		}
		return strings.TrimSpace(x)
	}
	return r.RemoteAddr
}

// RequestURL returns the full request URL as a string (scheme + host + path).
func RequestURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	s := r.URL.String()
	if r.URL.Scheme == "" {
		return "http://" + r.Host + s
	}
	return s
}
