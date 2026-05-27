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

## Upstreams with gRPC reflection <VersionTag version="v3.9.0" />

When the upstream **does** expose `grpc.reflection.v1.ServerReflection`, you can start GripMock with just the URL:

```bash
gripmock grpc+capture://orders.api.internal:8443
```

GripMock automatically:

1. Fetches service descriptors from the upstream via gRPC reflection
2. Registers services locally for proxying
3. Exposes them through its own gRPC reflection endpoint

This enables tools like `grpcurl` to discover services through GripMock without needing local `.proto` files:

```bash
# Discover services via GripMock's reflection
grpcurl localhost:4770 list

# Call proxied service
grpcurl -d '{"order_id":"123"}' localhost:4770 orders.OrderService.GetOrder
```

## Upstreams without gRPC reflection <VersionTag version="v3.13.0" />

If the upstream does **not** expose `grpc.reflection.v1.ServerReflection`, pass local descriptor sources via the `-S` flag:

```bash
gripmock -S ./proto/orders.proto grpc+capture://orders.api.internal:8443
```

The `-S` flag accepts `.proto`, `.protoset`, `.pb` files, directories, or BSR module references. When GripMock sees a local source alongside an upstream URL, it uses the local descriptors and never asks the upstream for its schema.

Service-to-upstream binding is derived from the local descriptor pool. No additional query parameter is required.

Example with imports:

```bash
gripmock -i ./proto -S ./proto/orders.proto grpc+capture://orders.api.internal:8443
```

## Per-proxy source binding <VersionTag version="v3.13.0" />

When using multiple upstream proxies, you can bind different local sources to specific proxies by placing `-S` flags before each proxy URL:

```bash
gripmock \
  -S ./proto/orders.proto grpc+proxy://orders.api.internal:8443 \
  -S ./proto/users.proto grpc+replay://users.api.internal:8444
```

Each `-S` flag binds only to the proxy URL that immediately follows it. Sources before a proxy are used for that proxy only; proxies without preceding `-S` use reflection.

Examples:

```bash
# up1 gets a.proto and b.proto, up2 uses reflection
gripmock -S a.proto -S b.proto grpc+proxy://up1:4111 grpc+proxy://up2:4222

# up1 uses reflection, up2 gets a.proto
gripmock grpc+proxy://up1:4111 -S a.proto grpc+proxy://up2:4222
```

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
