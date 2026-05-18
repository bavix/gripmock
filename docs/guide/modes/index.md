# Upstream Modes <VersionTag version="v3.9.0" />

Upstream modes define how GripMock handles requests when reflection sources are used.

⚠️ **EXPERIMENTAL FEATURE**: Upstream modes (`proxy`, `replay`, `capture`) are experimental and may change without notice.

## Modes at a glance

- `proxy`: pure reverse proxy through GripMock.
- `replay`: local stubs first, upstream fallback on real matcher miss.
- `capture`: replay behavior plus automatic recording of upstream misses.

## Why this matters

For a typical Order Service rollout, modes let you move in predictable stages:

1. Start with `proxy` to route all traffic through GripMock and inspect real calls.
2. Move to `replay` when you have initial stubs and still need upstream fallback.
3. Use `capture` to accelerate coverage and transition away from live dependency.

## Reflection vs mode

- Reflection source (`grpc://`, `grpcs://`) => how descriptors are loaded.
- Upstream mode (`+proxy`, `+replay`, `+capture`) => how runtime requests are resolved.

## Upstreams without gRPC reflection <VersionTag version="v3.13.0" />

If the upstream does **not** expose `grpc.reflection.v1.ServerReflection`, pass the schema yourself via any local descriptor source (`.proto`, `.protoset`, `.pb`, directory, or BSR module). When GripMock sees a local source alongside an upstream URL, it uses the local descriptors and never asks the upstream for its schema.

```bash
gripmock -i ./proto ./proto/orders.proto grpc+capture://orders.api.internal:8443
```

The URL still declares the upstream (`host:port`, mode, TLS, auth, timeout). Service-to-upstream binding is derived from the local descriptor pool. No additional flag or query parameter is required.

## URL schemes

- `grpc+proxy://host:port`
- `grpc+replay://host:port`
- `grpc+capture://host:port`
- `grpcs+proxy://host:port`
- `grpcs+replay://host:port`
- `grpcs+capture://host:port`

## Multi-source mode binding

When multiple sources are provided:

- the mode applies only to services registered from that source;
- if a service exists in multiple sources, first source wins for that service;
- later sources do not override already-bound services.

Example:

```bash
gripmock \
  grpc+proxy://proxy:123 \
  grpc+replay://proxy1:321 \
  grpc+capture://proxy2:444
```

If services overlap like `(greeter greeter1) (greeter1 greeter2) (greeter2 greeter3)`, final binding is:

- `greeter`, `greeter1` -> `proxy`
- `greeter2` -> `replay`
- `greeter3` -> `capture`

## Detailed pages

- [Proxy mode](/guide/modes/proxy)
- [Replay mode](/guide/modes/replay)
- [Capture mode](/guide/modes/capture)
