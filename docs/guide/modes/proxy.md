# Proxy Mode <VersionTag version="v3.9.0" />

`proxy` is pure reverse-proxy mode.

## Behavior

For unary and all streaming methods:

- Request is forwarded to upstream.
- Response, status, headers, and trailers are returned from upstream.
- Local stubs are not used for matching or response selection.

## URL schemes

- `grpc+proxy://host:port`
- `grpcs+proxy://host:port`

For descriptor loading options, see [Upstreams with gRPC reflection](/guide/modes/#upstreams-with-grpc-reflection-v3-9-0) (when upstream supports reflection) or [Upstreams without gRPC reflection](/guide/modes/#upstreams-without-grpc-reflection-v3-9-0) (when local descriptors are needed).

## Query parameters

| Parameter            | Default | Description                                   |
| -------------------- | ------- | --------------------------------------------- |
| `timeout`            | `5s`    | Timeout for upstream requests.                |
| `bearer`             | —       | Bearer token to include in upstream requests. |
| `serverName`         | —       | Override TLS server name (SNI).               |
| `insecureSkipVerify` | `false` | Skip upstream TLS certificate verification.   |
| `clientCert`         | —       | Client certificate presented to an upstream that requires mTLS. |
| `clientKey`          | —       | Private key for `clientCert`; both must be given together.      |
| `caFile`             | —       | CA that signs the upstream certificate (private PKI).           |

Example:

```bash
gripmock "grpcs+proxy://10.0.0.5:8443?serverName=api.company.local&timeout=10s"
```

## Upstream that requires mTLS <VersionTag version="v3.22.0" />

Point GripMock at the client certificate the upstream expects. The three file
parameters need a TLS scheme (`grpcs`) — on a plaintext upstream they are rejected
rather than ignored, so a connection never looks authenticated without being it:

```bash
gripmock "grpcs+proxy://orders.api.local:8443?clientCert=/certs/client.pem&clientKey=/certs/client.key&caFile=/certs/ca.pem"
```

The same parameters work for `grpc+capture://` and for a reflection source
(`grpcs://host:port`). A bad path fails at startup, not on the first proxied call.

Certificates belong to the URL, not to the process, so several upstreams behind
different PKIs work side by side:

```bash
gripmock \
  "grpcs+proxy://orders.api.local:8443?clientCert=/certs/orders.pem&clientKey=/certs/orders.key&caFile=/certs/orders-ca.pem" \
  "grpcs+proxy://billing.api.local:8443?clientCert=/certs/billing.pem&clientKey=/certs/billing.key&caFile=/certs/billing-ca.pem"
```

If one upstream rejects its certificate, GripMock stops at startup and names that
URL instead of serving half of the services.

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
