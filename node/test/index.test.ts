import { describe, it, expect, beforeEach } from 'vitest';
import { createServer } from 'node:http';
import { once } from 'node:events';
import type { AddressInfo } from 'node:net';
import { init, getTracer } from '../src/index.js';

describe('@permanu/sdk', () => {
 beforeEach(() => {
  for (const name of ['PERMANU_SERVICE_NAME','PERMANU_DEPLOYMENT_ID','OTEL_SERVICE_NAME','OTEL_RESOURCE_ATTRIBUTES','OTEL_EXPORTER_OTLP_ENDPOINT','OTEL_EXPORTER_OTLP_TRACES_ENDPOINT','OTEL_EXPORTER_OTLP_PROTOCOL','OTEL_EXPORTER_OTLP_TRACES_PROTOCOL']) delete process.env[name];
 });
 it('leaves unconfigured applications usable', async () => {
  await init()();
  getTracer().startSpan('no-op').end();
 });
 it('exports standard configuration to an owned HTTP protobuf collector once', async () => {
  const requests: { path?: string; contentType?: string; body: Buffer }[] = [];
  const collector = createServer((req,res) => {
   const chunks: Buffer[] = [];
   req.on('data', chunk => chunks.push(Buffer.from(chunk)));
   req.on('end', () => {
    requests.push({ path:req.url, contentType:req.headers['content-type'], body:Buffer.concat(chunks) });
    res.writeHead(200, {'content-type':'application/x-protobuf'}); res.end();
   });
  });
  collector.listen(0,'127.0.0.1'); await once(collector,'listening');
  try {
   process.env.OTEL_SERVICE_NAME='standard-service';
   process.env.PERMANU_SERVICE_NAME='ignored-legacy';
   process.env.OTEL_RESOURCE_ATTRIBUTES='permanu.project_id=project%2Cname,permanu.environment=prod%20east,permanu.service_id=svc,permanu.deployment_id=dep';
   process.env.OTEL_EXPORTER_OTLP_ENDPOINT=`http://127.0.0.1:${(collector.address() as AddressInfo).port}/prefix`;
   const shutdown = init();
   expect(init()).toBe(shutdown);
   getTracer().startSpan('fixture-span').end();
   await shutdown(); await shutdown();
   expect(requests).toHaveLength(1);
   expect(requests[0]?.path).toBe('/prefix/v1/traces');
   expect(requests[0]?.contentType).toBe('application/x-protobuf');
   const body=requests[0]!.body.toString();
   for (const value of ['standard-service','project,name','prod east','permanu.service_id','svc','permanu.deployment_id','dep','fixture-span']) expect(body).toContain(value);
   expect(body).not.toContain('ignored-legacy');
  } finally { collector.close(); await once(collector,'close'); }
 },10000);
});
