package permanu

import "testing"

func TestTelemetryConfiguration(t *testing.T) {
	for _, key := range []string{"OTEL_SERVICE_NAME", "OTEL_RESOURCE_ATTRIBUTES", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_PROTOCOL", "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "PERMANU_SERVICE_NAME", "PERMANU_DEPLOYMENT_ID"} {
		t.Setenv(key, "")
	}
	t.Setenv("PERMANU_SERVICE_NAME", "legacy")
	t.Setenv("OTEL_SERVICE_NAME", "standard")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "permanu.project_id=project%2Cname,permanu.environment=prod%20east,service.name=ignored,bad=%ZZ")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318/base/")
	config, ok := configuration()
	if !ok || config.service != "standard" || config.protocol != "http/protobuf" || config.endpoint != "http://collector:4318/base/v1/traces" {
		t.Fatalf("incorrect standard configuration: %#v", config)
	}
	if config.attrs["permanu.project_id"] != "project,name" || config.attrs["permanu.environment"] != "prod east" || config.attrs["service.name"] != "standard" {
		t.Fatal("resource attributes or precedence incorrect")
	}
	if _, exists := config.attrs["bad"]; exists {
		t.Fatal("malformed percent escape accepted")
	}
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://collector/custom")
	config, ok = configuration()
	if !ok || config.endpoint != "https://collector/custom" {
		t.Fatal("signal endpoint precedence lost")
	}
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "grpc")
	config, ok = configuration()
	if !ok || config.protocol != "grpc" {
		t.Fatal("explicit gRPC selection lost")
	}
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/json")
	if _, ok := configuration(); ok {
		t.Fatal("unsupported protocol accepted")
	}
}
