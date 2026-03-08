---
title: TLS and mTLS
---

# TLS and mTLS <VersionTag version="v3.8.1" />

GripMock supports native TLS for gRPC and optional mTLS (client certificate authentication).

## How TLS is enabled

gRPC TLS is enabled only when both variables are set:

- `GRPC_TLS_CERT_FILE`
- `GRPC_TLS_KEY_FILE`

If one of them is missing, gRPC starts without TLS.

## gRPC TLS environment variables

| Variable | Purpose | Required |
|---|---|---|
| `GRPC_TLS_CERT_FILE` | Server certificate file (PEM) | Yes, for TLS |
| `GRPC_TLS_KEY_FILE` | Server private key file (PEM) | Yes, for TLS |
| `GRPC_TLS_CLIENT_AUTH` | Enable mTLS (`true`/`false`) | No (default: `false`) |
| `GRPC_TLS_CA_FILE` | CA bundle used to verify client certificates | Required when `GRPC_TLS_CLIENT_AUTH=true` |
| `GRPC_TLS_MIN_VERSION` | Minimal TLS version (`1.2` or `1.3`) | No (default: `1.2`) |

## HTTP TLS environment variables

The HTTP API can also run with TLS:

| Variable | Purpose |
|---|---|
| `HTTP_TLS_CERT_FILE` | HTTP server certificate file |
| `HTTP_TLS_KEY_FILE` | HTTP server private key file |
| `HTTP_TLS_CLIENT_AUTH` | Enable mTLS for HTTP clients |
| `HTTP_TLS_CA_FILE` | CA bundle for HTTP client certificate verification |

## Examples

### gRPC TLS (server auth only)

```bash
GRPC_TLS_CERT_FILE=./third_party/tls/tls12/server.crt \
GRPC_TLS_KEY_FILE=./third_party/tls/tls12/server.key \
GRPC_TLS_CA_FILE=./third_party/tls/tls12/ca.crt \
GRPC_TLS_MIN_VERSION=1.2 \
gripmock --stub=examples examples
```

### gRPC mTLS

```bash
GRPC_TLS_CERT_FILE=./third_party/tls/mtls/server.crt \
GRPC_TLS_KEY_FILE=./third_party/tls/mtls/server.key \
GRPC_TLS_CA_FILE=./third_party/tls/mtls/ca.crt \
GRPC_TLS_CLIENT_AUTH=true \
GRPC_TLS_MIN_VERSION=1.3 \
gripmock --stub=examples examples
```

## `gripmock check` with TLS

`gripmock check` uses the same `GRPC_TLS_*` environment variables as the server.

For TLS-enabled gRPC, `check` must trust the server certificate chain.

### Minimal env for TLS (server auth only)

```bash
GRPC_TLS_CA_FILE=./third_party/tls/tls13/ca.crt \
GRPC_TLS_MIN_VERSION=1.3 \
gripmock check --timeout=60s --silent
```

### Env for mTLS

```bash
GRPC_TLS_CA_FILE=./third_party/tls/mtls/ca.crt \
GRPC_TLS_CERT_FILE=./third_party/tls/mtls/client.crt \
GRPC_TLS_KEY_FILE=./third_party/tls/mtls/client.key \
GRPC_TLS_MIN_VERSION=1.3 \
gripmock check --timeout=60s --silent
```

Notes:

- `GRPC_TLS_CA_FILE` is required in practice for self-signed/local CA certs.
- `GRPC_TLS_CERT_FILE` + `GRPC_TLS_KEY_FILE` are required only when server enforces mTLS.
- `GRPC_TLS_CLIENT_AUTH` affects server behavior and is not required for `check` itself.

### Client-side test env (grpctestify)

When running `grpctestify`, set these vars for TLS calls:

```bash
GRPCTESTIFY_TLS_CA_FILE=./third_party/tls/tls13/ca.crt
GRPCTESTIFY_TLS_SERVER_NAME=localhost
GRPCTESTIFY_TLS_CERT_FILE=./third_party/tls/mtls/client.crt
GRPCTESTIFY_TLS_KEY_FILE=./third_party/tls/mtls/client.key
```

For plain TLS (no mTLS), `GRPCTESTIFY_TLS_CERT_FILE` and `GRPCTESTIFY_TLS_KEY_FILE` can be empty.

## Reverse proxy TLS termination

If you prefer terminating TLS on a proxy, keep GripMock on an internal port and route traffic through Caddy/Nginx.

### Self-signed certificate setup

```bash
mkdir certs && openssl req \
  -x509 -newkey rsa:2048 \
  -keyout certs/key.pem -out certs/cert.pem \
  -days 365 -nodes \
  -subj "/CN=localhost"
```

### Caddy

```yaml
services:
  gripmock:
    # ... existing configuration ...

  caddy:
    image: caddy:latest
    ports:
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - ./certs:/certs
    depends_on:
      - gripmock
    # ... other configuration ...
```

```text
localhost:443 {
  tls /certs/cert.pem /certs/key.pem
  reverse_proxy gripmock:4770 {
    transport http {
      tls_insecure_skip_verify
    }
  }
}
```

### Nginx

```nginx
server {
  listen 443 ssl http2;
  ssl_certificate /etc/nginx/certs/cert.pem;
  ssl_certificate_key /etc/nginx/certs/key.pem;

  location / {
    grpc_pass grpc://gripmock:4770;
    grpc_ssl_verify off;
  }
}
```

### Verification

```bash
grpcurl -proto helloworld.proto \
  -cacert certs/cert.pem \
  -d '{"name": "TLS Test"}' \
  localhost:443 \
  helloworld.Greeter/SayHello
```
