import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { trace } from '@opentelemetry/api';

describe('@permanu/sdk', () => {
  beforeEach(() => {
    delete process.env['PERMANU_SERVICE_NAME'];
    delete process.env['PERMANU_DEPLOYMENT_ID'];
  });

  it('returns a no-op shutdown when PERMANU_SERVICE_NAME is unset', async () => {
    const { init } = await import('../src/index.ts');
    const shutdown = init();
    expect(typeof shutdown).toBe('function');
    await expect(shutdown()).resolves.toBeUndefined();
  });

  it('returns a callable tracer when PERMANU_SERVICE_NAME is unset', async () => {
    const { getTracer } = await import('../src/index.ts');
    const tracer = getTracer();
    expect(tracer).toBeDefined();
    // No-op tracer — starting a span must not throw.
    const span = tracer.startSpan('test-span');
    expect(() => span.end()).not.toThrow();
  });

  it('init() installs a provider and getTracer() returns a working tracer', async () => {
    process.env['PERMANU_SERVICE_NAME'] = 'test-service';
    process.env['PERMANU_DEPLOYMENT_ID'] = 'dep-123';

    // Dynamic import to pick up fresh module state with env set.
    const mod = await import('../src/index.ts?env=set');
    // Module is already loaded; call init directly.
    const { init, getTracer } = await import('../src/index.ts');
    const shutdown = init();
    expect(typeof shutdown).toBe('function');

    const tracer = getTracer();
    expect(tracer).toBeDefined();

    const span = tracer.startSpan('test-span');
    expect(() => span.end()).not.toThrow();

    await shutdown();
  });
});
