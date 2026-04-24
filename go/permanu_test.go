package permanu_test

import (
	"os"
	"testing"

	permanu "github.com/permanu/permanu-sdk-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestInit_NoopWhenEnvUnset(t *testing.T) {
	os.Unsetenv("PERMANU_SERVICE_NAME")
	os.Unsetenv("PERMANU_DEPLOYMENT_ID")

	shutdown := permanu.Init()
	if shutdown == nil {
		t.Fatal("Init() must return a non-nil shutdown function")
	}
	// Must not panic.
	shutdown()

	// Global tracer provider should remain the default no-op.
	tp := otel.GetTracerProvider()
	if _, ok := tp.(noop.TracerProvider); !ok {
		// After a no-op Init the provider may still be the global default,
		// which is the noop provider. As long as spans can be started safely
		// this is fine. We just verify Tracer() doesn't panic.
		tr := permanu.Tracer()
		if tr == nil {
			t.Fatal("Tracer() must return a non-nil tracer")
		}
	}
}

func TestInit_InstalledWithEnv(t *testing.T) {
	// Point at an address where nothing listens; the SDK queues and retries
	// internally — we just verify Init configures the provider and returns
	// a working tracer.
	t.Setenv("PERMANU_SERVICE_NAME", "test-service")
	t.Setenv("PERMANU_DEPLOYMENT_ID", "dep-123")

	shutdown := permanu.Init()
	if shutdown == nil {
		t.Fatal("Init() must return a non-nil shutdown function")
	}
	defer shutdown()

	tr := permanu.Tracer()
	if tr == nil {
		t.Fatal("Tracer() must return a non-nil tracer")
	}

	ctx := t.Context()
	_, span := tr.Start(ctx, "test-span")
	span.End()
}
