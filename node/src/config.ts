export interface TelemetryConfiguration {
  protocol: 'grpc' | 'http/protobuf';
  endpoint: string;
  attributes: Record<string, string>;
}

/** Runner variables take precedence; legacy host apps retain loopback gRPC. */
export function configuration(env: NodeJS.ProcessEnv): TelemetryConfiguration | undefined {
  const service = env.OTEL_SERVICE_NAME || env.PERMANU_SERVICE_NAME;
  if (!service) return undefined;
  const attributes: Record<string,string> = Object.create(null) as Record<string,string>;
  const raw = env.OTEL_RESOURCE_ATTRIBUTES || '';
  if (raw.length <= 64 * 1024) {
    for (const pair of raw.split(',').slice(0,64)) {
      const equals = pair.indexOf('=');
      if (equals < 1) continue;
      const key = pair.slice(0,equals).trim();
      if (!key) continue;
      try { attributes[key] = decodeURIComponent(pair.slice(equals+1).trim()); } catch { /* Ignore malformed attribute. */ }
    }
  }
  if (!attributes['permanu.deployment_id'] && env.PERMANU_DEPLOYMENT_ID) attributes['permanu.deployment_id'] = env.PERMANU_DEPLOYMENT_ID;
  attributes['service.name'] = service;
  const standard = Boolean(env.OTEL_SERVICE_NAME || env.OTEL_EXPORTER_OTLP_ENDPOINT || env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT);
  const protocol = env.OTEL_EXPORTER_OTLP_TRACES_PROTOCOL || env.OTEL_EXPORTER_OTLP_PROTOCOL || (standard ? 'http/protobuf' : 'grpc');
  if (protocol !== 'grpc' && protocol !== 'http/protobuf') return undefined;
  let endpoint = env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT;
  if (!endpoint) {
    endpoint = env.OTEL_EXPORTER_OTLP_ENDPOINT || (protocol === 'grpc' ? 'http://127.0.0.1:4317' : 'http://127.0.0.1:4318');
    if (protocol === 'http/protobuf') endpoint = endpoint.replace(/\/+$/, '') + '/v1/traces';
  }
  try {
    const url = new URL(endpoint);
    if (!['http:','https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) return undefined;
  } catch { return undefined; }
  return { protocol, endpoint, attributes };
}
