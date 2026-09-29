import permanu


def test_standard_configuration_and_percent_decoding(monkeypatch):
    for key in ("OTEL_SERVICE_NAME", "OTEL_RESOURCE_ATTRIBUTES", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_PROTOCOL", "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "PERMANU_SERVICE_NAME", "PERMANU_DEPLOYMENT_ID"):
        monkeypatch.delenv(key, raising=False)
    monkeypatch.setenv("OTEL_SERVICE_NAME", "standard")
    monkeypatch.setenv("PERMANU_SERVICE_NAME", "ignored")
    monkeypatch.setenv("OTEL_RESOURCE_ATTRIBUTES", "permanu.project_id=project%2Cname,permanu.environment=prod%20east,bad=%ZZ,service.name=ignored")
    monkeypatch.setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318/base/")
    protocol, endpoint, attrs = permanu._configuration()
    assert protocol == "http/protobuf"
    assert endpoint == "http://collector:4318/base/v1/traces"
    assert attrs["service.name"] == "standard"
    assert attrs["permanu.project_id"] == "project,name"
    assert attrs["permanu.environment"] == "prod east"
    assert "bad" not in attrs
    monkeypatch.setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://collector/custom")
    assert permanu._configuration()[1] == "https://collector/custom"
    monkeypatch.setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "grpc")
    assert permanu._configuration()[0] == "grpc"
    monkeypatch.setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/json")
    assert permanu._configuration() is None


def test_http_export_attribution_in_fresh_process():
    import os
    import subprocess
    import sys
    import threading
    from http.server import BaseHTTPRequestHandler, HTTPServer
    from opentelemetry.proto.collector.trace.v1.trace_service_pb2 import ExportTraceServiceRequest

    received = []

    class Collector(BaseHTTPRequestHandler):
        def do_POST(self):
            raw = self.rfile.read(int(self.headers["Content-Length"]))
            message = ExportTraceServiceRequest()
            message.ParseFromString(raw)
            received.append((self.path, self.headers["Content-Type"], message))
            self.send_response(200)
            self.send_header("Content-Type", "application/x-protobuf")
            self.end_headers()

        def log_message(self, *args):
            pass

    server = HTTPServer(("127.0.0.1", 0), Collector)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        env = dict(os.environ)
        for key in list(env):
            if key.startswith("OTEL_") or key.startswith("PERMANU_"):
                env.pop(key)
        env.update(OTEL_SERVICE_NAME="standard-service", PERMANU_SERVICE_NAME="ignored-legacy", OTEL_RESOURCE_ATTRIBUTES="permanu.project_id=project%2Cname,permanu.environment=prod%20east,permanu.service_id=svc,permanu.deployment_id=dep", OTEL_EXPORTER_OTLP_ENDPOINT=f"http://127.0.0.1:{server.server_port}/prefix")
        subprocess.run([sys.executable, "-c", "import permanu; shutdown=permanu.init(); assert permanu.init() is shutdown; span=permanu.get_tracer().start_span('fixture-span'); span.end(); shutdown(); shutdown()"], env=env, check=True, timeout=10)
        assert len(received) == 1
        path, content_type, message = received[0]
        assert path == "/prefix/v1/traces"
        assert content_type == "application/x-protobuf"
        attrs = {entry.key: entry.value.string_value for entry in message.resource_spans[0].resource.attributes}
        assert attrs["service.name"] == "standard-service"
        assert attrs["permanu.project_id"] == "project,name"
        assert attrs["permanu.environment"] == "prod east"
        assert attrs["permanu.service_id"] == "svc"
        assert attrs["permanu.deployment_id"] == "dep"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)
