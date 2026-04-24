# Permanu SDK

Customer instrumentation SDK for [Permanu](https://permanu.com)-deployed applications.

Permanu already captures ingress, container lifecycle, and deploy spans with zero code on your part. This SDK is for when you want your own app's internals — HTTP handlers, DB queries, cache lookups — nested under those platform spans in the trace viewer.

**One import. One init call. Done.**

---

## How it works

Every Permanu deploy host runs a local `pagent` process that accepts OTLP/gRPC on `127.0.0.1:4317`. The SDK configures OpenTelemetry to export there. Spans never leave the host via the SDK — they flow through `pagent` to Permanu's collector over the secure control plane.

Two environment variables are injected by Permanu at deploy time:

| Variable | Purpose |
|---|---|
| `PERMANU_SERVICE_NAME` | Sets `service.name` on all spans |
| `PERMANU_DEPLOYMENT_ID` | Sets `permanu.deployment_id` attribute |

**When these are unset** (local dev, CI, tests), every SDK silently no-ops. Leave the init call in place — it does nothing outside Permanu.

---

## Go

**Requirements:** Go 1.22+

```go
import permanu "github.com/permanu/permanu-sdk-go"

func main() {
    shutdown := permanu.Init()
    defer shutdown()

    tracer := permanu.Tracer()
    ctx, span := tracer.Start(ctx, "my-operation")
    defer span.End()

    // ... your code ...
}
```

**go.mod:**
```
require github.com/permanu/permanu-sdk-go v0.1.0
```

The SDK configures the global `otel.TracerProvider`. If you already use OTel directly, you can skip `permanu.Tracer()` and call `otel.Tracer("your-scope")` as usual — `Init()` just wires up the exporter.

---

## Node

**Requirements:** Node 18+ / Bun 1+

```typescript
import { init, getTracer } from '@permanu/sdk';

const shutdown = init();
process.on('SIGTERM', () => shutdown());

const tracer = getTracer();
const span = tracer.startSpan('my-operation');
// ... your code ...
span.end();
```

Or with the OTel context API for automatic parent/child linking:

```typescript
import { init, getTracer, context, trace } from '@permanu/sdk';

init();

const tracer = getTracer();
tracer.startActiveSpan('my-handler', (span) => {
    // nested spans created here will be children automatically
    span.end();
});
```

**package.json:**
```json
{
  "dependencies": {
    "@permanu/sdk": "^0.1.0"
  }
}
```

---

## Python

**Requirements:** Python 3.9+

```python
import permanu

shutdown = permanu.init()

tracer = permanu.get_tracer()
with tracer.start_as_current_span("my-operation") as span:
    span.set_attribute("db.query", "SELECT ...")
    # ... your code ...

# On process exit:
shutdown()
```

**pyproject.toml:**
```toml
[project]
dependencies = [
    "permanu-sdk>=0.1.0",
]
```

---

## Advanced use

The SDK does not wrap OTel — it configures it. You can use the full OTel API directly after calling `init()`:

```python
from opentelemetry import trace
from opentelemetry.trace import SpanKind

import permanu
permanu.init()

tracer = trace.get_tracer("my.library")
with tracer.start_as_current_span("db.query", kind=SpanKind.CLIENT) as span:
    span.set_attribute("db.system", "postgresql")
    span.set_attribute("db.statement", "SELECT ...")
```

All OTel semantic conventions apply. Permanu surfaces standard attributes (HTTP status, DB system, etc.) in the trace viewer automatically.

---

## Transport

- **Endpoint:** `127.0.0.1:4317` (loopback only, never external)
- **Protocol:** OTLP/gRPC, plaintext (loopback — TLS is not needed)
- **Batching:** default OTel batch settings; safe to leave as-is

No data leaves the host via this SDK. No secrets. No version telemetry.

---

## License

MIT — see [LICENSE](LICENSE).
