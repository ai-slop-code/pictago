package middleware

import (
	"net/http"

	"pictago/internal/requestctx"
)

// SecurityHeaders wraps next with common security headers. forUI adds CSP and X-Frame-Options for the admin UI.
func SecurityHeaders(next http.Handler, forUI bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if forUI {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://unpkg.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'")
		}
		next.ServeHTTP(w, r)
	})
}

// ClientIP sets the client IP on the request context (from X-Forwarded-For or RemoteAddr) and calls next.
func ClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := requestctx.WithClientIP(r.Context(), requestctx.ClientIPFromRequest(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
