// Package permanu configures application tracing from the runner's standard
// OpenTelemetry environment, with legacy PERMANU_* support for host applications.
package permanu

import (
	"context"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type telemetryConfig struct {
	service, protocol, endpoint string
	attrs                       map[string]string
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func configuration() (telemetryConfig, bool) {
	service := first(os.Getenv("OTEL_SERVICE_NAME"), os.Getenv("PERMANU_SERVICE_NAME"))
	if service == "" {
		return telemetryConfig{}, false
	}
	attrs := map[string]string{}
	raw := os.Getenv("OTEL_RESOURCE_ATTRIBUTES")
	if len(raw) <= 64*1024 {
		for index, pair := range strings.Split(raw, ",") {
			if index >= 64 {
				break
			}
			key, value, ok := strings.Cut(pair, "=")
			key = strings.TrimSpace(key)
			decoded, err := url.PathUnescape(strings.TrimSpace(value))
			if ok && key != "" && err == nil {
				attrs[key] = decoded
			}
		}
	}
	if id := os.Getenv("PERMANU_DEPLOYMENT_ID"); id != "" && attrs["permanu.deployment_id"] == "" {
		attrs["permanu.deployment_id"] = id
	}
	attrs["service.name"] = service
	protocol := first(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"), os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"))
	if protocol == "" {
		protocol = "grpc"
		if os.Getenv("OTEL_SERVICE_NAME") != "" || os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" {
			protocol = "http/protobuf"
		}
	}
	if protocol != "grpc" && protocol != "http/protobuf" {
		return telemetryConfig{}, false
	}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			if protocol == "grpc" {
				endpoint = "http://127.0.0.1:4317"
			} else {
				endpoint = "http://127.0.0.1:4318"
			}
		}
		if protocol == "http/protobuf" {
			endpoint = strings.TrimRight(endpoint, "/") + "/v1/traces"
		}
	}
	target, err := url.Parse(endpoint)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return telemetryConfig{}, false
	}
	return telemetryConfig{service, protocol, endpoint, attrs}, true
}

var initialization struct {
	sync.Mutex
	shutdown func()
}

// Init installs one provider per process. Repeated calls return the same
// idempotent shutdown function. Export and shutdown waits are bounded.
// Without a service name, it returns a no-op without changing global state.
func Init() func() {
	initialization.Lock()
	defer initialization.Unlock()
	if initialization.shutdown != nil {
		return initialization.shutdown
	}
	config, ok := configuration()
	if !ok {
		return func() {}
	}
	attrs := make([]attribute.KeyValue, 0, len(config.attrs))
	for key, value := range config.attrs {
		attrs = append(attrs, attribute.String(key, value))
	}
	res := resource.NewWithAttributes("", attrs...)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var exporter sdktrace.SpanExporter
	var err error
	if config.protocol == "grpc" {
		exporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(config.endpoint), otlptracegrpc.WithTimeout(3*time.Second))
	} else {
		exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(config.endpoint), otlptracehttp.WithTimeout(3*time.Second))
	}
	if err != nil {
		return func() {}
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter, sdktrace.WithExportTimeout(3*time.Second)), sdktrace.WithResource(res))
	otel.SetTracerProvider(provider)
	var once sync.Once
	initialization.shutdown = func() {
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = provider.Shutdown(ctx)
		})
	}
	return initialization.shutdown
}

// Tracer returns the application's tracer.
func Tracer() trace.Tracer { return otel.Tracer("app") }
