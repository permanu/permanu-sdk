# Permanu SDK

Customer instrumentation SDK for [Permanu](https://permanu.com)-deployed applications.

Permanu already captures ingress, container lifecycle, and deploy spans with zero code on your part. This SDK is for when you want your own app's internals — HTTP handlers, DB queries, cache lookups — nested under those platform spans in the trace viewer.

**One import. One init call. Done.**

---

## How it works

The runner injects the host-local OTLP receiver as `http://permanu-otel:4318` into application containers. The SDK exports HTTP protobuf traces to that endpoint and attaches the project, environment, service and deployment identities.

These standard environment variables are injected by Permanu at deploy time:

| Variable | Purpose |
|---|---|
| `OTEL_SERVICE_NAME` | Sets `service.name` on all spans |
| `OTEL_RESOURCE_ATTRIBUTES` | Percent-encoded project, environment, service and deployment attributes |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Base receiver URL; HTTP traces append `/v1/traces` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` |

Without `OTEL_SERVICE_NAME` or legacy `PERMANU_SERVICE_NAME`, initialization leaves the existing tracer provider unchanged. Legacy host applications using `PERMANU_SERVICE_NAME` and `PERMANU_DEPLOYMENT_ID` retain the loopback gRPC default. Standard service names take precedence.

---

## Go

**Requirements:** Go 1.24+

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

Signal-specific `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` and `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL` override their generic counterparts. A signal-specific HTTP URL is used as given; a generic HTTP URL appends `/v1/traces`. Explicit `grpc` remains supported. Unsupported protocols or invalid endpoints leave tracing unchanged.

Initialization creates one provider per process. Shutdown is idempotent and waits at most five seconds; individual exports use a three-second timeout. Attribute parsing retains at most 64 entries from at most 64 KiB, decodes percent escapes, and skips malformed entries. Applications configuring their own endpoint are responsible for that destination.

---

## License

MIT — see [LICENSE](LICENSE).
