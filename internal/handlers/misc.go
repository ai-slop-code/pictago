package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"pictago/internal/auth"
)

func (s *Server) configHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer("pictago").Start(r.Context(), "config")
		defer span.End()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"max_upload_bytes": s.MaxUploadBytes,
			"max_upload_mb":    s.MaxUploadBytes >> 20,
		})
	}
}

func (s *Server) statsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "stats")
		defer span.End()
		r = r.WithContext(ctx)
		username := auth.UsernameFromRequest(r)
		if username == "" {
			span.SetStatus(codes.Error, "unauthorized")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := s.Auth.GetUserID(r.Context(), username)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "user not found")
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		st, err := s.Auth.GetStats(r.Context(), userID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "get stats")
			http.Error(w, "failed to get stats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	}
}

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("pictago").Start(r.Context(), "ui.serve")
	defer span.End()
	r = r.WithContext(ctx)
	data, err := fs.ReadFile(s.UI, "index.html")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read index")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate nonce")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nonceStr := base64.StdEncoding.EncodeToString(nonce)
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://cdn.jsdelivr.net 'nonce-"+nonceStr+"' 'unsafe-eval'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := string(data)
	html = strings.Replace(html, "__CSP_NONCE__", nonceStr, 1)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("serveUI write: %v", err)
	}
}
