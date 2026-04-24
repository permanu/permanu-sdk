// Package permanu is the customer SDK for Permanu-deployed applications.
// One-line setup:
//
//	func main() {
//	    shutdown := permanu.Init()
//	    defer shutdown()
//	    // ... your code; use permanu.Tracer() for spans ...
//	}
//
// When running on a Permanu host, spans flow to the local pagent OTLP
// receiver on 127.0.0.1:4317 and render in Permanu's trace viewer nested
// under the ingress/deploy spans. When running outside Permanu
// (local dev, tests), Init returns a no-op and Tracer returns a no-op
// tracer — safe to leave in place.
package permanu

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Init configures OpenTelemetry to export spans to the local pagent receiver.
// If PERMANU_SERVICE_NAME is unset (e.g. local dev), Init is a no-op and
// returns a safe no-op shutdown function.
func Init() func() {
	serviceName := os.Getenv("PERMANU_SERVICE_NAME")
	if serviceName == "" {
		return func() {}
	}

	deploymentID := os.Getenv("PERMANU_DEPLOYMENT_ID")

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
	}
	if deploymentID != "" {
		attrs = append(attrs, attribute.String("permanu.deployment_id", deploymentID))
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		// Resource build failure is non-fatal; fall back to no-op.
		return func() {}
	}

	exp, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint("127.0.0.1:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return func() {}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return func() {
		_ = tp.Shutdown(context.Background())
	}
}

// Tracer returns a package-level tracer named "app".
// When running outside Permanu (Init was a no-op), this returns a no-op tracer.
func Tracer() trace.Tracer {
	return otel.Tracer("app")
}
