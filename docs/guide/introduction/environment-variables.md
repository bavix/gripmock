# Environment Variables

GripMock reads configuration from environment variables on startup.

## Core

| Variable | Default | Description |
|---|---|---|
| `LOG_LEVEL` | `info` | Log level (`trace`, `debug`, `info`, `warn`, `error`). |
| `MAX_NESTING_DEPTH` | `256` | Max message nesting depth during stub matching (safety net for circular refs). |

## gRPC server

| Variable | Default | Description |
|---|---|---|
| `GRPC_NETWORK` | `tcp` | Network type for gRPC listener. |
| `GRPC_HOST` | `0.0.0.0` | gRPC bind host. |
| `GRPC_PORT` | `4770` | gRPC bind port. |
| `GRPC_ADDR` | `$GRPC_HOST:$GRPC_PORT` | Full gRPC bind address. |

## gRPC limits and keepalive <VersionTag version="v3.19.0" />

Sizes accept a `K`/`M`/`G` suffix or plain bytes. Defaults reproduce the values
used before these variables existed, so an unset environment changes nothing.

| Variable | Default | Description |
|---|---|---|
| `GRPC_MAX_RECV_MSG_SIZE` | `4M` | Largest accepted inbound message. A client sending more gets `RESOURCE_EXHAUSTED`. |
| `GRPC_MAX_SEND_MSG_SIZE` | `0` | Largest outbound message; `0` means unlimited. |
| `GRPC_KEEPALIVE_TIME` | `30s` | Idle period before the server pings the client. |
| `GRPC_KEEPALIVE_TIMEOUT` | `10s` | How long to wait for a ping reply before closing the connection. |
| `GRPC_KEEPALIVE_MAX_CONNECTION_IDLE` | `5m` | Close a connection idle for this long. |
| `GRPC_KEEPALIVE_MAX_CONNECTION_AGE` | `30m` | Close a connection this old, regardless of traffic. |

These apply to the gRPC port only. The ConnectRPC/gRPC-web gateway caps a frame
at 4 MiB independently, so raising `GRPC_MAX_RECV_MSG_SIZE` does not lift the
limit for gateway traffic.

## HTTP admin server

| Variable | Default | Description |
|---|---|---|
| `HTTP_HOST` | `0.0.0.0` | HTTP bind host (admin API + UI). |
| `HTTP_PORT` | `4771` | HTTP bind port. |
| `HTTP_ADDR` | `$HTTP_HOST:$HTTP_PORT` | Full HTTP bind address. |
| `CORS_ALLOWED_ORIGINS` | `*` | Origins allowed to call the admin API and the gateway. Also decides which origins may reach `/api/mcp`: a browser origin that is neither loopback nor listed here is refused. |
| `CORS_ALLOWED_METHODS` | `GET,POST,DELETE,PATCH` | Methods allowed for cross-origin requests. |

## Gateway server <VersionTag version="v3.17.0" />

The gateway serves both **ConnectRPC** and **gRPC-web** protocols on a single HTTP port. Content-Type negotiation dispatches to the correct handler automatically.

| Variable | Default | Description |
|---|---|---|
| `GATEWAY_HOST` | `0.0.0.0` | Gateway bind host. |
| `GATEWAY_PORT` | `4769` | Gateway bind port. |
| `GATEWAY_ADDR` | `$GATEWAY_HOST:$GATEWAY_PORT` | Full gateway bind address. |
| `GATEWAY_TLS_CERT_FILE` | *(empty)* | Gateway server TLS certificate file. |
| `GATEWAY_TLS_KEY_FILE` | *(empty)* | Gateway server TLS private key file. |
| `GATEWAY_TLS_CLIENT_AUTH` | `false` | Require client certs for gateway (mTLS). |
| `GATEWAY_TLS_CA_FILE` | *(empty)* | CA file for validating gateway client certs. |
| `GATEWAY_TLS_MIN_VERSION` | `1.2` | Minimum TLS version (`1.2`, `1.3`). |
| `CONNECT_REQUIRE_PROTOCOL_VERSION` <VersionTag version="v3.19.0" /> | `false` | Reject Connect requests without `Connect-Protocol-Version: 1` (or `?connect=v1` on GET) with `400`. |

The gateway provides unary and streaming RPC support for both protocols over HTTP/1.1 and HTTP/2 (with or without TLS). It shares the same stub storage, descriptor registry, and history store as gRPC and REST servers.

::: warning Removed in v3.20.0
The `CONNECTRPC_HOST` / `CONNECTRPC_PORT` / `CONNECTRPC_ADDR` and `CONNECTRPC_TLS_*`
fallbacks are gone. Use the `GATEWAY_*` variables above.
:::

## Stub watcher

| Variable | Default | Description |
|---|---|---|
| `STUB_WATCHER_ENABLED` | `true` | Enable automatic file watch/reload for stubs. |
| `STUB_WATCHER_INTERVAL` | `1s` | Polling interval for timer-based watcher. |
| `STUB_WATCHER_TYPE` | `fsnotify` | Watcher backend (`fsnotify`, `timer`). |

## History

| Variable | Default | Description |
|---|---|---|
| `HISTORY_ENABLED` | `true` | Enable call history recording. |
| `HISTORY_LIMIT` | `64M` | In-memory history size cap. |
| `HISTORY_MESSAGE_MAX_BYTES` | `262144` | Max stored payload size per message. |
| `HISTORY_REDACT_KEYS` | *(empty)* | Comma-separated keys to redact in history. |

## Session GC

| Variable | Default | Description |
|---|---|---|
| `SESSION_GC_INTERVAL` | `30s` | Session cleanup loop interval. |
| `SESSION_GC_TTL` | `60s` | Session time-to-live. |

## Plugins

| Variable | Default | Description |
|---|---|---|
| `TEMPLATE_PLUGIN_PATHS` | *(empty)* | Comma-separated paths to template plugins. |

## gRPC TLS

| Variable | Default | Description |
|---|---|---|
| `GRPC_TLS_CERT_FILE` | *(empty)* | gRPC server TLS certificate file. |
| `GRPC_TLS_KEY_FILE` | *(empty)* | gRPC server TLS private key file. |
| `GRPC_TLS_CLIENT_AUTH` | `false` | Require client certs for gRPC (mTLS). |
| `GRPC_TLS_CA_FILE` | *(empty)* | CA file for validating gRPC client certs. |
| `GRPC_TLS_MIN_VERSION` | `1.2` | Minimum TLS version (`1.2`, `1.3`). |

## HTTP TLS

| Variable | Default | Description |
|---|---|---|
| `HTTP_TLS_CERT_FILE` | *(empty)* | HTTP server TLS certificate file. |
| `HTTP_TLS_KEY_FILE` | *(empty)* | HTTP server TLS private key file. |
| `HTTP_TLS_CLIENT_AUTH` | `false` | Require client certs for HTTP (mTLS). |
| `HTTP_TLS_CA_FILE` | *(empty)* | CA file for validating HTTP client certs. |
| `HTTP_TLS_MIN_VERSION` | `1.2` | Minimum TLS version (`1.2`, `1.3`). |

## OpenTelemetry

| Variable | Default | Description |
|---|---|---|
| `OTEL_ENABLED` | `false` | Enable OpenTelemetry export. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP collector endpoint. |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | Use insecure OTLP transport. |

## Buf Schema Registry (BSR)

Supported profiles:

- `BSR_BUF_*`
- `BSR_SELF_*`

Variables per profile:

| Variable suffix | Default | Description |
|---|---|---|
| `BASE_URL` | *(empty)* | BSR API base URL. |
| `TOKEN` | *(empty)* | BSR token (private modules). |
| `TIMEOUT` | `5s` | BSR request timeout. |

Examples:

- `BSR_BUF_BASE_URL`, `BSR_BUF_TOKEN`, `BSR_BUF_TIMEOUT`
- `BSR_SELF_BASE_URL`, `BSR_SELF_TOKEN`, `BSR_SELF_TIMEOUT`

## Notes for CLI utilities

### dump

`gripmock dump` reads `HTTP_ADDR` for the admin API host/port.

- Default scheme is `http`. Use `--scheme https` for TLS.
- Override address via env: `HTTP_ADDR=10.0.0.5:4771 gripmock dump`.
