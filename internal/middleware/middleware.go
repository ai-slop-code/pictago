package middleware

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

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

// RequestID sets or generates a request ID, adds it to context and to the response header X-Request-ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			b := make([]byte, 8)
			if _, _ = rand.Read(b); true {
				id = hex.EncodeToString(b)
			}
		}
		w.Header().Set("X-Request-ID", id)
		ctx := requestctx.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// gzipResponseWriter wraps the response and compresses with gzip when appropriate.
type gzipResponseWriter struct {
	http.ResponseWriter
	req *http.Request
	gz  *gzip.Writer
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.gz == nil {
		ct := g.Header().Get("Content-Type")
		accept := g.req.Header.Get("Accept-Encoding")
		if strings.Contains(accept, "gzip") && (strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/html")) {
			g.Header().Set("Content-Encoding", "gzip")
			g.Header().Del("Content-Length")
			g.gz = gzip.NewWriter(g.ResponseWriter)
		}
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(p []byte) (n int, err error) {
	if g.gz != nil {
		return g.gz.Write(p)
	}
	return g.ResponseWriter.Write(p)
}

func (g *gzipResponseWriter) closeGz() {
	if g.gz != nil {
		_ = g.gz.Close()
		g.gz = nil
	}
}

// Gzip compresses responses with gzip for JSON and HTML when the client accepts it.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gw := &gzipResponseWriter{ResponseWriter: w, req: r}
		defer gw.closeGz()
		next.ServeHTTP(gw, r)
	})
}
