# Environment Variables

GripMock reads configuration from environment variables on startup.

## Core

| Variable | Default | Description |
|---|---|---|
| `LOG_LEVEL` | `info` | Log level (`trace`, `debug`, `info`, `warn`, `error`). |

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
