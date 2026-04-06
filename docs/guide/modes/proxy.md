# Proxy Mode <VersionTag version="v3.9.0" />

`proxy` is pure reverse-proxy mode.

⚠️ **EXPERIMENTAL FEATURE**: `proxy` mode is part of the experimental upstream modes feature set and may change without notice.

## Behavior

For unary and all streaming methods:

- Request is forwarded to upstream.
- Response, status, headers, and trailers are returned from upstream.
- Local stubs are not used for matching or response selection.

## URL schemes

- `grpc+proxy://host:port`
- `grpcs+proxy://host:port`

## Query parameters

| Parameter | Default | Description |
|---|---|---|
| `timeout` | `5s` | Timeout for upstream requests. |
| `bearer` | — | Bearer token to include in upstream requests. |
| `serverName` | — | Override TLS server name (SNI). |
| `insecureSkipVerify` | `false` | Skip upstream TLS certificate verification. |

Example:

```bash
gripmock "grpcs+proxy://10.0.0.5:8443?serverName=api.company.local&timeout=10s"
```

## Order Service example

```bash
GRPC_PORT=4770 HTTP_PORT=4771 \
gripmock "grpcs+proxy://orders.api.local:8443?serverName=orders.api.local"
```

Point your application/test environment to `localhost:4770`.
GripMock forwards every call and logs request/response (`gRPC call completed`), which is useful for baseline traffic inspection before creating stubs.

## When to choose `proxy`

- You need immediate startup with no stub preparation.
- You want reverse-proxy behavior only.
- You want real traffic visibility in GripMock logs.
