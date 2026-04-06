# Replay Mode <VersionTag version="v3.9.0" />

`replay` is local-first mode with upstream fallback.

⚠️ **EXPERIMENTAL FEATURE**: `replay` mode is part of the experimental upstream modes feature set and may change without notice.

## Behavior

For each incoming request:

1. GripMock tries local stub matching.
2. If a local match is found, local response is returned.
3. If matcher returns `NotFound`, request is forwarded to upstream.

This applies to unary, server-stream, client-stream, and bidirectional stream methods.

## Matching semantics

Fallback is based on real matcher result with full criteria:

- `input` / `inputs`
- `headers`
- `session`
- `contains` / `matches`
- `options.times`

No heuristic shortcut is used for fallback decisions.

## URL schemes

- `grpc+replay://host:port`
- `grpcs+replay://host:port`

## Query parameters

| Parameter | Default | Description |
|---|---|---|
| `timeout` | `5s` | Timeout for upstream requests. |
| `bearer` | — | Bearer token to include in upstream requests. |
| `serverName` | — | Override TLS server name (SNI). |
| `insecureSkipVerify` | `false` | Skip upstream TLS certificate verification. |
| `recordDelay` | `false` | Record response latency as `delay` in captured stubs (for upstream misses). |

## Example

```bash
GRPC_PORT=4770 HTTP_PORT=4771 \
gripmock "grpcs+replay://orders.api.local:8443?serverName=orders.api.local"
```

## Order Service example

You already have minimal stubs for critical flows (for example `CreateOrder`, `GetOrder`), but many methods are still uncovered.

Runtime effect:

- requests that match local stubs are served by GripMock;
- unmatched requests fallback to real Order Service.

This mode is intended for safe gradual migration.

## When to choose `replay`

- You already have partial local stubs.
- You still need upstream reliability for uncovered paths.
- You want deterministic behavior for covered paths with incremental expansion.
