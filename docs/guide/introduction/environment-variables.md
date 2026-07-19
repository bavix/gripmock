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

## HTTP admin server

| Variable | Default | Description |
|---|---|---|
| `HTTP_HOST` | `0.0.0.0` | HTTP bind host (admin API + UI). |
| `HTTP_PORT` | `4771` | HTTP bind port. |
| `HTTP_ADDR` | `$HTTP_HOST:$HTTP_PORT` | Full HTTP bind address. |

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

The gateway provides unary and streaming RPC support for both protocols over HTTP/1.1 and HTTP/2 (with or without TLS). It shares the same stub storage, descriptor registry, and history store as gRPC and REST servers.

### Legacy aliases (deprecated)

The following `CONNECTRPC_*` variables are still supported as fallbacks when the corresponding `GATEWAY_*` variable is not set:

| Deprecated | Unified |
|---|---|
| `CONNECTRPC_HOST` | `GATEWAY_HOST` |
| `CONNECTRPC_PORT` | `GATEWAY_PORT` |
| `CONNECTRPC_ADDR` | `GATEWAY_ADDR` |
| `CONNECTRPC_TLS_CERT_FILE` | `GATEWAY_TLS_CERT_FILE` |
| `CONNECTRPC_TLS_KEY_FILE` | `GATEWAY_TLS_KEY_FILE` |
| `CONNECTRPC_TLS_CLIENT_AUTH` | `GATEWAY_TLS_CLIENT_AUTH` |
| `CONNECTRPC_TLS_CA_FILE` | `GATEWAY_TLS_CA_FILE` |
| `CONNECTRPC_TLS_MIN_VERSION` | `GATEWAY_TLS_MIN_VERSION` |

These aliases will be removed in a future release. Migrate to `GATEWAY_*` variables.

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
