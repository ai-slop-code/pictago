// Package telemetry provides OpenTelemetry tracing for export to Elastic APM Server (or any OTLP backend).
package telemetry

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	defaultServiceName = "pictago"
	shutdownTimeout    = 10 * time.Second
)

// Init initializes the global tracer provider with an OTLP HTTP exporter.
// endpoint is the OTLP endpoint (e.g. http://localhost:8200 for Elastic APM Server, or http://localhost:4318 for a generic collector).
// If endpoint is empty, a no-op provider is set and shutdown does nothing.
// Call the returned shutdown function before process exit (e.g. defer).
func Init(ctx context.Context, endpoint, serviceName string) (shutdown func(), err error) {
	if endpoint == "" {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		return func() {}, nil
	}

	endpoint = strings.TrimSpace(endpoint)
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if host == "" {
		host = endpoint
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(host),
		otlptracehttp.WithInsecure(), // use WithTLSCertificate for production TLS
	}
	if u.Path != "" && u.Path != "/" {
		opts = append(opts, otlptracehttp.WithURLPath(u.Path))
	}

	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	if serviceName == "" {
		serviceName = defaultServiceName
	}
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		_ = exp.Shutdown(ctx)
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	shutdown = func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = tp.Shutdown(shutdownCtx)
	}
	return shutdown, nil
}
