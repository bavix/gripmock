# ConnectRPC <VersionTag version="v3.15.0" />

ConnectRPC is a modern HTTP RPC protocol. GripMock supports it out of the box, including unary, server-streaming, client-streaming, and bidirectional streaming over the standard Connect envelope framing.

The gateway serves ConnectRPC and gRPC-web on a **single port** (`:4769` by default). Content-Type negotiation dispatches to the correct handler automatically. See [gRPC-web](grpc-web) for the companion protocol.

## HTTP Interface

```
POST /{service}/{method}
```

## Content Types

| Content-Type | Format |
|---|---|
| `application/json` | JSON (unary) |
| `application/proto` | Protobuf binary (unary) |
| `application/connect+json` | JSON, Connect envelope framing (streaming) |
| `application/connect+proto` | Protobuf binary, Connect envelope framing (streaming) |

## Wire Format

### Unary

Request and response bodies are **raw** protobuf or JSON — no binary framing. Errors use non-200 HTTP status codes with a JSON error body:

```http
HTTP/1.1 404 Not Found
Content-Type: application/connect+json

{"code":"not_found","message":"method not found","details":[]}
```

### Streaming

Each message is wrapped in a 5-byte Connect envelope. The flag bit `0x02` marks the final message (end-of-stream):

```
┌─────────┬──────────────────┐
│ flags   │ length (BE u32)  │ payload bytes
└─────────┴──────────────────┘
```

## Request Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | Determines serialization (see table above) |
| `Content-Encoding` | `gzip`, `deflate`, `zstd`, `snappy`, or `br` |
| `Accept-Encoding` | `gzip` or `deflate` (response compression) |
| `X-Gripmock-Session` | Session ID for call tracking |

## Examples

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice"}'
```

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/proto" \
  --data-binary @request.pb
```

## Stub Configuration

Stubs work identically across all protocols.

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

## Features

- **Input Matching** — `equals`, `contains`, `regex`, `glob`, `anyOf`
- **Output Templates** — `faker.*`, `{{uuid}}`, `{{timestamp}}`
- **Delay Simulation** — `output.delay`
- **Error Responses** — `output.error` with gRPC status codes
- **Headers** — Custom metadata in responses
- **Health Checks** — Via `/grpc.health.v1.Health/Check`
- **Streaming** — Unary, server-streaming, client-streaming, bidirectional
- **Request Compression** — `gzip`, `deflate`, `zstd`, `snappy`, `brotli`
- **Response Compression** — `gzip`, `deflate`
- **OpenTelemetry** — Tracing and metrics

## TLS

Configured via `GATEWAY_TLS_*` variables. See [Environment Variables](/guide/introduction/environment-variables).

## Version History

| Version | Change |
|---|---|
| v3.15.0 | ConnectRPC server on a dedicated port (`CONNECTRPC_PORT`) |
| v3.17.0 | Unified gateway: ConnectRPC + gRPC-web on a single port (`GATEWAY_PORT`) |

## Related

- [gRPC-web](grpc-web)
- [Environment Variables](/guide/introduction/environment-variables)
- [Quick Start](/guide/introduction/quick-usage)
