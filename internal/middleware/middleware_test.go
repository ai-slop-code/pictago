package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pictago/internal/requestctx"
)

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := SecurityHeaders(next, false)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options not set")
	}
}

func TestSecurityHeaders_forUI(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := SecurityHeaders(next, true)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options not set for UI")
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("CSP not set for UI")
	}
}

func TestGzip_compressesJSON(t *testing.T) {
	body := `{"key":"value"}`
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})
	handler := Gzip(next)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestGzip_noGzipWithoutAccept(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})
	handler := Gzip(next)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("Content-Encoding should be empty, got %q", rec.Header().Get("Content-Encoding"))
	}
	if string(rec.Body.Bytes()) != "{}" {
		t.Errorf("body = %q", rec.Body.Bytes())
	}
}

func TestClientIP_setsContext(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := requestctx.ClientIP(r.Context())
		// Middleware sets whatever ClientIPFromRequest returns (RemoteAddr when no X-Forwarded-For)
		if ip != "1.2.3.4:5678" {
			t.Errorf("ClientIP = %q", ip)
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := ClientIP(next)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}
