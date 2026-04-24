/**
 * @permanu/sdk — One-line OpenTelemetry setup for Permanu-deployed apps.
 *
 * Usage:
 *   import { init, getTracer } from '@permanu/sdk';
 *   const shutdown = init();
 *   process.on('SIGTERM', () => shutdown());
 *
 * When PERMANU_SERVICE_NAME is unset (local dev / tests), init() is a no-op
 * and getTracer() returns the global no-op tracer. Safe to leave in place.
 *
 * Spans are exported to 127.0.0.1:4317 (the local pagent OTLP receiver).
 * Nothing ever leaves the host.
 */

import { context, trace, type Tracer } from '@opentelemetry/api';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { Resource } from '@opentelemetry/resources';
import { NodeTracerProvider, BatchSpanProcessor } from '@opentelemetry/sdk-trace-node';
import { ATTR_SERVICE_NAME } from '@opentelemetry/semantic-conventions';

export type ShutdownFn = () => Promise<void>;

/**
 * Configures OTel to export spans to the local pagent receiver.
 * Returns an async shutdown function — call it on process exit.
 *
 * No-ops when PERMANU_SERVICE_NAME is not set.
 */
export function init(): ShutdownFn {
  const serviceName = process.env['PERMANU_SERVICE_NAME'];
  if (!serviceName) {
    return async () => {};
  }

  const deploymentId = process.env['PERMANU_DEPLOYMENT_ID'];

  const resourceAttrs: Record<string, string> = {
    [ATTR_SERVICE_NAME]: serviceName,
  };
  if (deploymentId) {
    resourceAttrs['permanu.deployment_id'] = deploymentId;
  }

  const resource = new Resource(resourceAttrs);

  const exporter = new OTLPTraceExporter({
    url: 'http://127.0.0.1:4317',
  });

  const provider = new NodeTracerProvider({
    resource,
    spanProcessors: [new BatchSpanProcessor(exporter)],
  });

  provider.register();

  return async () => {
    await provider.shutdown();
  };
}

/**
 * Returns a tracer named "app".
 * When init() was a no-op, this returns the global no-op tracer.
 */
export function getTracer(): Tracer {
  return trace.getTracer('app');
}

// Re-export OTel context for convenience.
export { context, trace };
