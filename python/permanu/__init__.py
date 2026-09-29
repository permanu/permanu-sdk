"""Application tracing from standard OTEL_* or legacy PERMANU_* variables."""
from __future__ import annotations

import os
import re
import threading
from typing import Callable
from urllib.parse import unquote, urlsplit

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter as HTTPSpanExporter

_lock = threading.Lock()
_shutdown: Callable[[], None] | None = None


def _configuration() -> tuple[str, str, dict[str, str]] | None:
    service = os.environ.get("OTEL_SERVICE_NAME") or os.environ.get("PERMANU_SERVICE_NAME")
    if not service:
        return None
    attrs: dict[str, str] = {}
    raw = os.environ.get("OTEL_RESOURCE_ATTRIBUTES", "")
    if len(raw) <= 64 * 1024:
        for pair in raw.split(",")[:64]:
            key, sep, value = pair.partition("=")
            key, value = key.strip(), value.strip()
            if sep and key and not re.search(r"%(?![0-9a-fA-F]{2})", value):
                try:
                    attrs[key] = unquote(value, errors="strict")
                except UnicodeDecodeError:
                    pass
    legacy_id = os.environ.get("PERMANU_DEPLOYMENT_ID", "")
    if legacy_id and not attrs.get("permanu.deployment_id"):
        attrs["permanu.deployment_id"] = legacy_id
    attrs["service.name"] = service
    standard = any(os.environ.get(key) for key in ("OTEL_SERVICE_NAME", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"))
    protocol = os.environ.get("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL") or os.environ.get("OTEL_EXPORTER_OTLP_PROTOCOL") or ("http/protobuf" if standard else "grpc")
    if protocol not in ("grpc", "http/protobuf"):
        return None
    endpoint = os.environ.get("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
    if not endpoint:
        endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT") or ("http://127.0.0.1:4317" if protocol == "grpc" else "http://127.0.0.1:4318")
        if protocol == "http/protobuf":
            endpoint = endpoint.rstrip("/") + "/v1/traces"
    try:
        target = urlsplit(endpoint)
        if target.scheme not in ("http", "https") or not target.hostname or target.username or target.password or target.query or target.fragment:
            return None
        _ = target.port
    except ValueError:
        return None
    return protocol, endpoint, attrs


def init() -> Callable[[], None]:
    """Install one provider and return a bounded, idempotent shutdown callable.

    An unconfigured application keeps its existing global tracer provider.
    """
    global _shutdown
    with _lock:
        if _shutdown is not None:
            return _shutdown
        config = _configuration()
        if config is None:
            return lambda: None
        protocol, endpoint, attrs = config
        if protocol == "grpc":
            exporter = OTLPSpanExporter(endpoint=endpoint, insecure=endpoint.startswith("http://"), timeout=3)
        else:
            exporter = HTTPSpanExporter(endpoint=endpoint, timeout=3)
        provider = TracerProvider(resource=Resource(attributes=attrs))
        provider.add_span_processor(BatchSpanProcessor(exporter, export_timeout_millis=3000))
        trace.set_tracer_provider(provider)
        shutdown_lock = threading.Lock()
        pending: threading.Thread | None = None

        def shutdown() -> None:
            nonlocal pending
            with shutdown_lock:
                if pending is None:
                    pending = threading.Thread(target=provider.shutdown, name="permanu-telemetry-shutdown", daemon=True)
                    pending.start()
                thread = pending
            thread.join(timeout=5)
            if thread.is_alive():
                raise TimeoutError("Telemetry shutdown timed out")

        _shutdown = shutdown
        return shutdown


def get_tracer(name: str = "app") -> trace.Tracer:
    """Return an application tracer; safe in unconfigured processes."""
    return trace.get_tracer(name)
