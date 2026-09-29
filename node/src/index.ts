/** One-line tracing from standard OTEL_* variables or legacy PERMANU_* names. */
import { context, trace, type Tracer } from '@opentelemetry/api';
import { OTLPTraceExporter as GRPCExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { OTLPTraceExporter as HTTPExporter } from '@opentelemetry/exporter-trace-otlp-proto';
import { Resource } from '@opentelemetry/resources';
import { NodeTracerProvider, BatchSpanProcessor } from '@opentelemetry/sdk-trace-node';
import { configuration } from './config.js';

export type ShutdownFn = () => Promise<void>;
let shutdown: ShutdownFn | undefined;

/** Installs one provider; repeated calls share one bounded, idempotent shutdown. */
export function init(): ShutdownFn {
  if (shutdown) return shutdown;
  const config = configuration(process.env);
  if (!config) return async () => {};
  const options = { url:config.endpoint, timeoutMillis:3000 };
  const exporter = config.protocol === 'grpc' ? new GRPCExporter(options) : new HTTPExporter(options);
  const provider = new NodeTracerProvider({
    resource: new Resource(config.attributes),
    spanProcessors: [new BatchSpanProcessor(exporter, { exportTimeoutMillis:3000 })],
  });
  provider.register();
  let pending: Promise<void> | undefined;
  shutdown = () => {
    if (!pending) {
      pending = new Promise<void>((resolve,reject) => {
        const timer = setTimeout(() => reject(new Error('Telemetry shutdown timed out')),5000);
        timer.unref();
        void provider.shutdown().then(resolve,reject).finally(() => clearTimeout(timer));
      });
    }
    return pending;
  };
  return shutdown;
}

export function getTracer(): Tracer { return trace.getTracer('app'); }
export { context, trace };
