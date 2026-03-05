package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"pictago/internal/audit"
	"pictago/internal/auth"
	"pictago/internal/config"
	"pictago/internal/handlers"
	"pictago/internal/middleware"
	"pictago/internal/storage"
	"pictago/internal/telemetry"
)

// version is set at build time via -ldflags "-X main.version=..."
var version string

//go:embed ui
var embedUI embed.FS

func main() {
	ctx := context.Background()

	otelEndpoint := config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", config.GetEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""))
	otelServiceName := config.GetEnv("OTEL_SERVICE_NAME", "pictago")
	shutdown, err := telemetry.Init(ctx, otelEndpoint, otelServiceName)
	if err != nil {
		log.Fatalf("telemetry: %v", err)
	}
	defer shutdown()

	dataDir := config.GetEnv("DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	dataDir, err = filepath.Abs(dataDir)
	if err != nil {
		log.Fatalf("resolve data dir: %v", err)
	}

	filesDir := config.GetEnv("FILES_DIR", filepath.Join(dataDir, "files"))
	dbPath := config.GetEnv("DB_PATH", filepath.Join(dataDir, "users.db"))
	auditLogPath := config.GetEnv("AUDIT_LOG_PATH", filepath.Join(dataDir, "audit.json"))
	maxUploadMB := config.GetEnvInt("MAX_UPLOAD_MB", 10)
	maxUploadBytes := int64(maxUploadMB) << 20

	auditLog, err := audit.New(auditLogPath)
	if err != nil {
		log.Fatalf("audit log: %v", err)
	}
	defer auditLog.Close()

	authStore, err := auth.NewStore(dbPath)
	if err != nil {
		log.Fatalf("auth store: %v", err)
	}
	defer authStore.Close()

	fileStore, err := storage.New(filesDir)
	if err != nil {
		log.Fatalf("file store: %v", err)
	}

	thumbDir := filepath.Join(dataDir, "thumbnails")
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		log.Fatalf("create thumbnails dir: %v", err)
	}

	uiFS, err := fs.Sub(embedUI, "ui")
	if err != nil {
		log.Fatalf("ui fs: %v", err)
	}

	server := handlers.NewServer(authStore, fileStore, auditLog, maxUploadBytes, thumbDir, uiFS)
	if v := config.GetEnv("VERSION", version); v != "" {
		server.Version = v
	} else {
		server.Version = "dev"
	}
	server.EnsureUser()

	addr := config.GetEnv("ADDR", ":8080")
	if auditLogPath != "" {
		log.Printf("audit log: %s", auditLogPath)
	} else {
		log.Printf("audit log: disabled")
	}
	if otelEndpoint != "" {
		log.Printf("APM: sending traces to %s (service %s)", otelEndpoint, otelServiceName)
	} else {
		log.Printf("APM: disabled (set OTEL_EXPORTER_OTLP_ENDPOINT to enable)")
	}
	log.Printf("listening on %s (max upload %d MiB)", addr, maxUploadMB)

	handler := middleware.RequestID(middleware.ClientIP(middleware.Gzip(server.Routes())))
	handler = otelhttp.NewHandler(handler, otelServiceName)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second, // allow slow uploads (e.g. 10 MB)
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Print("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Print("stopped")
}
