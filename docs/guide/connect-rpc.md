# ConnectRPC <VersionTag version="v3.15.0" />

::: warning Experimental Feature
The ConnectRPC server is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk and avoid in production-critical systems without careful consideration.
:::

ConnectRPC is a modern HTTP/2 RPC protocol that works alongside gRPC. GripMock supports ConnectRPC out of the box, including unary, server-streaming, client-streaming, and bidirectional streaming over the standard Connect envelope framing.

## HTTP Interface

```
POST /{service}/{method}
Content-Type: application/json  # or application/proto
```

## Content Types

| Content-Type | Format |
|-------------|--------|
| `application/json` | JSON (unary) |
| `application/proto` | Protobuf binary (unary) |
| `application/connect+json` | JSON, Connect envelope framing (streaming) |
| `application/connect+proto` | Protobuf binary, Connect envelope framing (streaming) |

ConnectRPC negotiates the wire format from the request's `Content-Type`. For unary calls the response uses the same family as the request; for streaming calls the response uses the Connect streaming variant (`application/connect+...`).

## Streaming Protocol

For streaming RPCs each message is wrapped in a 5-byte Connect envelope:

```
┌─────────┬──────────────────┐
│ flags   │ length (BE u32)  │ payload bytes
└─────────┴──────────────────┘
```

The high bit of `flags` is set on the final message to mark end-of-stream.

## Request Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json`, `application/proto`, or `application/connect+...` |
| `Content-Encoding` | `gzip`, `deflate`, `zstd`, `snappy`, or `br` (request body is decompressed before stub matching) |
| `Accept-Encoding` | `gzip` or `deflate` (response body is compressed when supported) |
| `X-Gripmock-Session` | Session ID for call tracking |

## Examples

### JSON (unary)

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice"}'
```

### Proto Binary (unary)

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/proto" \
  --data-binary @request.pb
```

### Streaming (gRPC client)

```go
stream, _ := client.StreamMethod(ctx, &Request{})
for {
    msg, err := stream.Recv()
    if errors.Is(err, io.EOF) { break }
    // process msg
}
```

The same `StreamingMethod` definition works for both gRPC and ConnectRPC transports — choose the client transport at connection time.

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

### With Streaming and Per-Event Delays

```yaml
service: TrackService
method: StreamTrack
input:
  equals:
    stn: "MS#00005"
output:
  stream:
    - data: { stn: "MS#00005", identity: "00" }
      delay: 100ms
    - data: { stn: "MS#0000502", identity: "01" }
      delay: 50ms
    - data: { stn: "MS#00005", identity: "02" }
```

Each `delay` applies before the corresponding message; elements without `delay` fall back to the uniform `output.delay` if set.

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
- **Delay Simulation** — `output.delay` and per-event `delay` on stream elements
- **Error Responses** — `output.error` with gRPC status codes
- **Headers** — Custom metadata in responses
- **Health Checks** — Via `/grpc.health.v1.Health/Check`
- **Streaming** — Unary, server-streaming, client-streaming, and bidirectional over Connect envelope framing
- **Request Compression** — `gzip`, `deflate`, `zstd`, `snappy`, `brotli` (Content-Encoding)
- **Response Compression** — `gzip`, `deflate` (Accept-Encoding, via gorilla/handlers.CompressHandler)
- **OpenTelemetry** — Tracing and metrics via `OTEL_ENABLED` and `otelhttp`

## TLS

ConnectRPC server supports native TLS via `CONNECTRPC_TLS_CERT_FILE`, `CONNECTRPC_TLS_KEY_FILE`, and optionally `CONNECTRPC_TLS_CA_FILE` with `CONNECTRPC_TLS_CLIENT_AUTH=true` for mTLS. The minimum supported TLS version is 1.2. The server speaks HTTP/1.1 and HTTP/2 on the same listener (negotiated via ALPN for TLS, or via h2c upgrade for plain HTTP).

## Compression

ConnectRPC supports two-way compression:

- **Request body** — clients may send `Content-Encoding: gzip | deflate | zstd | snappy | br` and GripMock decompresses before stub matching. Invalid encodings return `400 Bad Request`.
- **Response body** — when the client sends `Accept-Encoding: gzip` or `deflate`, GripMock compresses the response. Streaming responses are flushed after each message.

## Comparison

| Feature | gRPC | ConnectRPC |
|---------|------|------------|
| Browser support | Requires gRPC-Web | Direct |
| JSON | Via codec | Native |
| Streaming | Full support | Full support (envelope framing) |
| TLS | Configurable minimum version | Minimum 1.2 |
| Request compression | gRPC encoding | gzip, deflate, zstd, snappy, brotli |
| Response compression | gRPC encoding | gzip, deflate (HTTP) |

## Related

- [Quick Start](/guide/introduction/quick-usage)
- [Dynamic Templates](/guide/stubs/dynamic-templates)
- [Health Checks](/guide/stubs/health)
- [Streaming Stubs](/guide/stubs/streaming)
- [Stub Delays](/guide/stubs/delay)
