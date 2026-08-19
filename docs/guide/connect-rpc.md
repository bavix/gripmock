# ConnectRPC <VersionTag version="v3.15.0" />

ConnectRPC is an HTTP RPC protocol. GripMock serves it with the same stubs it serves gRPC from — unary, server-streaming, client-streaming and bidirectional, over the standard Connect envelope framing.

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
Content-Type: application/json

{"code":"not_found","message":"method not found"}
```

Error bodies are always `application/json`, whatever codec the request used. A
content type GripMock has no codec for is answered with `415 Unsupported Media Type`.

`output.details` are carried the way the protocol defines them — the type name plus
the serialized message, base64 without padding — so a Connect client can rebuild
them. The readable rendering is repeated in the optional `debug` field:

```json
{
  "code": "invalid_argument",
  "message": "bad id",
  "details": [{
    "type": "google.rpc.ErrorInfo",
    "value": "CglJRF9JTlZBTElE",
    "debug": {"@type": "type.googleapis.com/google.rpc.ErrorInfo", "reason": "ID_INVALID"}
  }]
}
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
| `Content-Encoding` | `gzip`, `deflate`, `zstd`, `snappy`, or `br` (unary) |
| `Accept-Encoding` | `gzip` or `deflate` (unary response compression) |
| `Connect-Content-Encoding` | Streaming compression. Only `identity` is supported; anything else answers `unimplemented` |
| `Connect-Timeout-Ms` | Per-call deadline; exceeding it answers `deadline_exceeded` |
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

- **Input Matching** — `equals`, `contains`, `matches`, `glob`, `anyOf`
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
| v3.20.0 | `CONNECTRPC_*` environment fallbacks removed |

## Related

- [gRPC-web](grpc-web)
- [Environment Variables](/guide/introduction/environment-variables)
- [Quick Start](/guide/introduction/quick-usage)
