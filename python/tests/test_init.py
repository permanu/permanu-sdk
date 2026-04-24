"""Tests for the permanu SDK."""

import os
import pytest
from unittest.mock import patch


def test_init_noop_when_env_unset(monkeypatch):
    """init() returns a safe no-op when PERMANU_SERVICE_NAME is unset."""
    monkeypatch.delenv("PERMANU_SERVICE_NAME", raising=False)
    monkeypatch.delenv("PERMANU_DEPLOYMENT_ID", raising=False)

    import permanu
    shutdown = permanu.init()
    assert callable(shutdown)
    # Must not raise.
    shutdown()


def test_get_tracer_noop_outside_permanu(monkeypatch):
    """get_tracer() returns a usable tracer even when init() was a no-op."""
    monkeypatch.delenv("PERMANU_SERVICE_NAME", raising=False)

    import permanu
    tracer = permanu.get_tracer()
    assert tracer is not None
    with tracer.start_as_current_span("test-span") as span:
        assert span is not None


def test_init_installs_provider(monkeypatch):
    """init() installs a real TracerProvider when PERMANU_SERVICE_NAME is set."""
    monkeypatch.setenv("PERMANU_SERVICE_NAME", "test-service")
    monkeypatch.setenv("PERMANU_DEPLOYMENT_ID", "dep-456")

    # Patch the exporter so no real gRPC connection is attempted.
    with patch(
        "permanu.OTLPSpanExporter",
        autospec=True,
    ) as mock_exporter_cls:
        mock_exporter_cls.return_value.export = lambda spans: None

        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry import trace

        import importlib
        import permanu as p
        importlib.reload(p)

        shutdown = p.init()
        assert callable(shutdown)

        provider = trace.get_tracer_provider()
        assert isinstance(provider, TracerProvider)

        tracer = p.get_tracer()
        assert tracer is not None
        with tracer.start_as_current_span("test-span") as span:
            assert span is not None

        shutdown()


def test_init_no_deployment_id(monkeypatch):
    """init() works without PERMANU_DEPLOYMENT_ID."""
    monkeypatch.setenv("PERMANU_SERVICE_NAME", "svc-no-depid")
    monkeypatch.delenv("PERMANU_DEPLOYMENT_ID", raising=False)

    with patch("permanu.OTLPSpanExporter", autospec=True):
        import importlib
        import permanu as p
        importlib.reload(p)

        shutdown = p.init()
        shutdown()
