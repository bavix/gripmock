# gRPC-web <VersionTag version="v3.17.0" />

gRPC-web is a browser-compatible protocol that brings gRPC to web applications. GripMock supports gRPC-web unary and streaming calls, with the same stubs and features as ConnectRPC and native gRPC.

The gateway serves gRPC-web and ConnectRPC on a **single port** (`:4769` by default). Content-Type negotiation dispatches to the correct handler automatically. See [ConnectRPC](connect-rpc) for the companion protocol.

## HTTP Interface

```
POST /{service}/{method}
```

## Content Types

| Content-Type | Format |
|---|---|
| `application/grpc-web+json` | JSON (length-prefixed + trailers) |
| `application/grpc-web+proto` | Protobuf binary (length-prefixed + trailers) |

## Wire Format

All messages are **length-prefixed** using a 5-byte envelope:

```
[1 byte flags][4 byte length (BE u32)][payload bytes]
```

| Flag | Meaning |
|---|---|
| `0x00` | Uncompressed data frame |
| `0x01` | Compressed data frame (not supported; use `Content-Encoding: gzip`) |
| `0x80` | Trailers frame (final frame with `grpc-status` / `grpc-message`) |

### Unary

Response consists of a data frame followed by a trailers frame. HTTP status is always `200 OK`; the actual gRPC status is carried in the trailers:

```http
HTTP/1.1 200 OK
Content-Type: application/grpc-web+proto

[0x00][4-byte len][proto message]
[0x80][4-byte len][grpc-status: 0\r\n]
```

### Errors

Errors are communicated via trailers in a trailers-only response:

```http
HTTP/1.1 200 OK
Content-Type: application/grpc-web+proto

[0x80][4-byte len][grpc-status: 5\r\ngrpc-message: not found\r\n]
```

### Streaming

Same envelope framing. Multiple data frames followed by a final trailers frame.

## Request Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/grpc-web+json` or `application/grpc-web+proto` |
| `Content-Encoding` | `gzip`, `deflate`, `zstd`, `snappy`, or `br` |
| `Accept-Encoding` | `gzip` or `deflate` (response compression) |
| `X-Gripmock-Session` | Session ID for call tracking |

## Examples

### JSON (unary)

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/grpc-web+json" \
  -d '{"name":"Alice"}'
```

### Proto Binary (unary)

```bash
curl -X POST http://localhost:4769/test.TestService/TestMethod \
  -H "Content-Type: application/grpc-web+proto" \
  --data-binary @request.pb
```

### Unsupported: text encoding

`application/grpc-web-text+*` (base64) is not supported. Use `application/grpc-web+proto` or `application/grpc-web+json` instead.

## Stub Configuration

Stubs work identically across all protocols. See [ConnectRPC](connect-rpc#stub-configuration) for examples.

## Features

All features from ConnectRPC apply to gRPC-web as well — input matching, templates, delays, errors, headers, streaming, compression, and OpenTelemetry.

The key difference: gRPC-web always uses HTTP `200 OK` (actual status in trailers), while ConnectRPC returns non-200 HTTP codes for errors.

## TLS

Configured via `GATEWAY_TLS_*` variables. See [Environment Variables](/guide/introduction/environment-variables).

## Comparison

| Feature | gRPC-web | ConnectRPC | gRPC |
|---|---|---|---|
| Browser support | Direct | Direct | Requires proxy |
| Unary response format | Length-prefixed + trailers | Raw body | Length-prefixed |
| HTTP status on error | Always 200 | Non-200 | Always 200 |
| Error location | Trailers frame | JSON body | Trailers |
| JSON support | Native | Native | Via codec |
| Streaming | Full (envelope) | Full (envelope) | Full |

## Related

- [ConnectRPC](connect-rpc)
- [Environment Variables](/guide/introduction/environment-variables)
- [Quick Start](/guide/introduction/quick-usage)
