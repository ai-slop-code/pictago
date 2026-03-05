package telemetry

import (
	"context"
	"testing"
)

func TestInit_emptyEndpoint(t *testing.T) {
	ctx := context.Background()
	shutdown, err := Init(ctx, "", "test-svc")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown func should be non-nil")
	}
	// No-op shutdown should not panic
	shutdown()
}
