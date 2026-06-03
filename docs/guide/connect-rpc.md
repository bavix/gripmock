# ConnectRPC <VersionTag version="v3.14.0" />

ConnectRPC is a modern HTTP/2 RPC protocol that works alongside gRPC. GripMock supports ConnectRPC out of the box.

## HTTP Interface

```
POST /{service}/{method}
Content-Type: application/json  # or application/proto
```

## Content Types

| Content-Type | Format |
|-------------|--------|
| `application/json` | JSON |
| `application/proto` | Protobuf binary |

## Request Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` or `application/proto` |
| `X-Gripmock-Session` | Session ID for call tracking |
| `Content-Encoding: gzip` | Compress request body |

## Examples

### JSON

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice"}'
```

### Proto Binary

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/proto" \
  --data-binary @request.pb
```

## Stub Configuration

Stubs work identically for gRPC and ConnectRPC.

```yaml
service: test.TestService
method: TestMethod
input:
  equals:
    name: "Alice"
output:
  data:
    greeting: "Hello, Alice!"
```

### With Delay

```yaml
service: test.TestService
method: TestMethod
input:
  equals:
    name: "Alice"
output:
  delay: 100ms
  data:
    greeting: "Hello, Alice!"
```

### With Error

```yaml
service: test.TestService
method: TestMethod
input:
  equals:
    name: "error"
output:
  error: "Simulated error"
  code: 13  # INTERNAL
```

### With Templates

```yaml
service: test.TestService
method: TestMethod
input:
  contains:
    name: "{{faker.Name}}"
output:
  data:
    id: "{{uuid}}"
    email: "{{faker.Email}}"
```

## Session Tracking

Set the `X-Gripmock-Session` header to group requests:

```bash
curl -X POST http://localhost:4769/test.Service/Method \
  -H "X-Gripmock-Session: test-session-123" \
  -H "Content-Type: application/json" \
  -d '{}'
```

Sessions are recorded in call history for later verification.

## Features

All gRPC features work with ConnectRPC:

- **Input Matching** — `equals`, `contains`, `regex`, `glob`, `anyOf`
- **Output Templates** — `faker.*`, `{{uuid}}`, `{{timestamp}}`
- **Delay Simulation** — `output.delay`
- **Error Responses** — `output.error` with gRPC status codes
- **Headers** — Custom metadata in responses
- **Health Checks** — Via `/grpc.health.v1.Health/Check`

## Limitations

### Streaming <VersionTag version="3.14.0" />

Streaming RPCs (`ClientStreaming`, `ServerStreaming`, `Bidirectional`) are **not supported** via ConnectRPC. Use gRPC for streaming calls.

### TLS

ConnectRPC works over plain HTTP. For secure connections, place GripMock behind a TLS-terminating proxy (nginx, Envoy, etc.).

## Comparison

| Feature | gRPC | ConnectRPC |
|---------|------|------------|
| Browser support | Requires gRPC-Web | Direct |
| JSON | Via codec | Native |
| Streaming | Full support | Not supported |

## Related

- [Quick Start](../introduction/quick-usage)
- [Dynamic Templates](../stubs/dynamic-templates)
- [Health Checks](../stubs/health)
