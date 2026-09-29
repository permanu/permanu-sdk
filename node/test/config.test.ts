import { it, expect } from 'vitest';
import { configuration } from '../src/config.js';

it('honors endpoint and protocol precedence, decodes attributes and preserves legacy gRPC', () => {
 const env = { OTEL_SERVICE_NAME:'standard', PERMANU_SERVICE_NAME:'legacy', OTEL_RESOURCE_ATTRIBUTES:'service.name=ignored,permanu.project_id=project%2Cname,permanu.environment=prod%20east,bad=%ZZ', OTEL_EXPORTER_OTLP_ENDPOINT:'http://collector:4318/base/' };
 const config=configuration(env)!;
 expect(config.protocol).toBe('http/protobuf');
 expect(config.endpoint).toBe('http://collector:4318/base/v1/traces');
 expect(config.attributes['service.name']).toBe('standard');
 expect(config.attributes['permanu.project_id']).toBe('project,name');
 expect(config.attributes['permanu.environment']).toBe('prod east');
 expect(config.attributes['bad']).toBeUndefined();
 expect(configuration({...env, OTEL_EXPORTER_OTLP_TRACES_ENDPOINT:'https://collector/custom'})?.endpoint).toBe('https://collector/custom');
 expect(configuration({...env, OTEL_EXPORTER_OTLP_TRACES_PROTOCOL:'grpc'})?.protocol).toBe('grpc');
 expect(configuration({...env, OTEL_EXPORTER_OTLP_TRACES_PROTOCOL:'http/json'})).toBeUndefined();
 expect(configuration({PERMANU_SERVICE_NAME:'legacy'})?.endpoint).toBe('http://127.0.0.1:4317');
 expect(configuration({PERMANU_SERVICE_NAME:'legacy'})?.protocol).toBe('grpc');
 expect(configuration({...env, OTEL_EXPORTER_OTLP_TRACES_ENDPOINT:'https://user:password@collector/custom'})).toBeUndefined();
});
