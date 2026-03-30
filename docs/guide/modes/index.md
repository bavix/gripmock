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
