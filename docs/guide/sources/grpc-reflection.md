# gRPC Reflection Source <VersionTag version="v3.9.0" />

GripMock can load API descriptors directly from a running gRPC server via reflection.

Need forwarding behavior (`proxy`, `replay`, `capture`) on top of reflection sources?
See [Upstream Modes](/guide/modes/index).

## Basic usage

Use an insecure endpoint:

```bash
gripmock grpc://localhost:50051
```

Use a TLS endpoint:

```bash
gripmock grpcs://api.company.local:443
```

With stubs:

```bash
gripmock --stub ./stubs grpc://localhost:50051
```

## URL format

- `grpc://host:port` -> no TLS
- `grpcs://host:port` -> TLS

GripMock reads descriptors from reflection API and then starts mock services from those descriptors.

## Query parameters

Supported query parameters:

- `timeout` (default: `5s`)
- `bearer` (Authorization token)
- `serverName` (TLS server name override, mostly for `grpcs://`)

Examples:

```bash
# Timeout
gripmock grpc://localhost:50051?timeout=10s

# Bearer token
gripmock grpc://localhost:50051?bearer=my-token

# TLS + SNI override
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
```

## Requirements on target server

Target gRPC server must:

1. Expose gRPC reflection (`grpc.reflection.v1`).
2. Be reachable from GripMock process.

If reflection is disabled, GripMock cannot load descriptors from this source.

## Notes

- Reflection and health services are skipped when building descriptor set.
- `bearer` is sent as `Authorization: Bearer <token>`.
- Current limitations: no custom CA, mTLS, or authority flags yet.
