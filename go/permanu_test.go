package permanu_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	permanu "github.com/permanu/permanu-sdk-go"
	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestInit_NoopWhenEnvUnset(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("PERMANU_SERVICE_NAME", "")
	permanu.Init()()
	_, span := permanu.Tracer().Start(t.Context(), "no-op")
	span.End()
}

func TestInit_ExportsAttributedHTTPProtobufAndShutdownIsIdempotent(t *testing.T) {
	requests := make(chan *collector.ExportTraceServiceRequest, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prefix/v1/traces" || r.Header.Get("Content-Type") != "application/x-protobuf" {
			t.Error("incorrect export route or encoding")
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		var message collector.ExportTraceServiceRequest
		if err != nil || proto.Unmarshal(raw, &message) != nil {
			t.Error("invalid protobuf export")
		}
		requests <- &message
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("OTEL_SERVICE_NAME", "standard-service")
	t.Setenv("PERMANU_SERVICE_NAME", "ignored-legacy")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL+"/prefix")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "permanu.project_id=project%2Cname,permanu.environment=prod%20east,permanu.service_id=svc,permanu.deployment_id=dep")
	shutdown := permanu.Init()
	_ = permanu.Init()
	_, span := permanu.Tracer().Start(t.Context(), "fixture-span")
	span.End()
	shutdown()
	shutdown()
	select {
	case message := <-requests:
		if len(message.ResourceSpans) != 1 {
			t.Fatal("missing resource spans")
		}
		attrs := map[string]string{}
		for _, item := range message.ResourceSpans[0].Resource.Attributes {
			attrs[item.Key] = item.Value.GetStringValue()
		}
		for key, want := range map[string]string{"service.name": "standard-service", "permanu.project_id": "project,name", "permanu.environment": "prod east", "permanu.service_id": "svc", "permanu.deployment_id": "dep"} {
			if attrs[key] != want {
				t.Errorf("incorrect attribute %s", key)
			}
		}
	default:
		t.Fatal("export did not reach owned collector")
	}
	select {
	case <-requests:
		t.Fatal("duplicate initialization or shutdown exported twice")
	default:
	}
}
