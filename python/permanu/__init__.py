"""Permanu customer SDK — one-line OpenTelemetry setup.

Usage:
    import permanu

    shutdown = permanu.init()
    # ... use permanu.get_tracer() for spans ...
    shutdown()

When running on a Permanu host, spans flow to the local pagent OTLP
receiver on 127.0.0.1:4317 and render in Permanu's trace viewer nested
under the ingress/deploy spans. When PERMANU_SERVICE_NAME is unset
(local dev, CI), init() is a no-op and get_tracer() returns the global
no-op tracer — safe to leave in place.

Nothing ever leaves the host; the exporter targets a loopback address only.
"""

from __future__ import annotations

import os
from typing import Callable

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource, SERVICE_NAME
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter


_ENDPOINT = "http://127.0.0.1:4317"


def init() -> Callable[[], None]:
    """Configure OTel to export spans to the local pagent receiver.

    Returns a zero-argument shutdown callable — call it at process exit.
    No-ops when PERMANU_SERVICE_NAME is not set.
    """
    service_name = os.environ.get("PERMANU_SERVICE_NAME", "")
    if not service_name:
        return lambda: None

    deployment_id = os.environ.get("PERMANU_DEPLOYMENT_ID", "")

    attributes = {SERVICE_NAME: service_name}
    if deployment_id:
        attributes["permanu.deployment_id"] = deployment_id

    resource = Resource(attributes=attributes)

    exporter = OTLPSpanExporter(endpoint=_ENDPOINT, insecure=True)
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)

    def shutdown() -> None:
        provider.shutdown()

    return shutdown


def get_tracer(name: str = "app") -> trace.Tracer:
    """Return a tracer for the given instrumentation scope.

    Defaults to "app". When init() was a no-op, returns the global
    no-op tracer — all span operations are safe but do nothing.
    """
    return trace.get_tracer(name)
