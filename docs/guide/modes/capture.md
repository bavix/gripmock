# Capture Mode <VersionTag version="v3.9.0" />

`capture` is replay mode plus automatic recording of upstream misses.

⚠️ **EXPERIMENTAL FEATURE**: `capture` mode is part of the experimental upstream modes feature set and may change without notice.

## Behavior

For each request:

1. Try local stub match first.
2. If local match exists, return local response.
3. If no local match, forward request to upstream.
4. Record upstream response/error as a local stub.

Recorded stubs are immediately available for later local replay.

## What gets recorded

Capture stores request and upstream result in stub form:

- request input (`input`/`inputs`)
- request headers (filtered user headers)
- response data/stream
- grpc error code/message/details (when present)
- response headers
- session context (if present)

## URL schemes

- `grpc+capture://host:port`
- `grpcs+capture://host:port`

## Query parameters

| Parameter | Default | Description |
|---|---|---|
| `timeout` | `5s` | Timeout for upstream requests. |
| `bearer` | — | Bearer token to include in upstream requests. |
| `serverName` | — | Override TLS server name (SNI). |
| `insecureSkipVerify` | `false` | Skip upstream TLS certificate verification. |
| `recordDelay` | `false` | Record response latency as `delay` in captured stubs. |

Example with delay recording enabled:

```bash
GRPC_PORT=4770 HTTP_PORT=4771 gripmock "grpc+capture://orders.api.local:4770?recordDelay=true"
```

## Order Service example

Goal: quickly mock `OrderService` with minimal manual stub authoring.

Start capture mode against the real upstream:

```bash
GRPC_PORT=4770 HTTP_PORT=4771 gripmock "grpcs+capture://orders.api.local:8443?serverName=orders.api.local"
```

Then point your application/test environment to `localhost:4770` and run normal workflows.

Runtime effect:

- matched local stubs are reused;
- unmatched requests are proxied to upstream;
- upstream responses/errors are recorded into GripMock storage.

After enough traffic, you get broad local coverage and can switch environments to replay/local-first behavior.

## When to choose `capture`

- You need fastest bootstrap for large or legacy services.
- You want to convert real traffic into reusable stubs.
- You want to replace upstream dependency progressively with local mocks.
