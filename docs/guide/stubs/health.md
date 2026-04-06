# Health Service Stubbing <VersionTag version="v3.9.3" />

GripMock supports stubbing the standard gRPC health service:

- `grpc.health.v1.Health/Check`
- `grpc.health.v1.Health/Watch`

This is useful when you want to test client behavior for dependency health transitions (for example `NOT_SERVING -> SERVING`).

## Protected internal key

The service key `gripmock` is reserved for GripMock internal readiness and cannot be mocked.

- `service: "gripmock"` always returns the real server health.
- Stubs targeting `gripmock` may be stored, but they are ignored at runtime.

## Behavior matrix

| Request `service` | Stub exists | Result |
|---|---|---|
| `gripmock` | Yes/No | Real server health (stub ignored) |
| Custom name | Yes | Mocked response |
| Custom name | No | Default health behavior (`Check` returns `NotFound`) |
| Empty `""` | Yes | Mocked response (like any custom name) |
| Empty `""` | No | Default health behavior |

## Example: `Check`

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

- service: grpc.health.v1.Health
  method: Check
  input:
    equals:
      service: examples.health.backend
  output:
    data:
      status: NOT_SERVING
```

Request:

```json
{
  "service": "examples.health.backend"
}
```

For `service: "gripmock"`, any stub is intentionally ignored and the real runtime status is returned.

Response:

```json
{
  "status": "NOT_SERVING"
}
```

## Example: `Watch` with delay

```yaml
- service: grpc.health.v1.Health
  method: Watch
  input:
    equals:
      service: examples.health.watch
  output:
    delay: 10ms
    stream:
      - status: NOT_SERVING
      - status: SERVING
```

The stream responses are sent in order with the configured delay.

## SDK example

See the [Embedded SDK — Health Checks](../embedded-sdk/health.md) for a full SDK-based health stub example.

## Full runnable example

See:

- `examples/projects/health/stubs.yaml`
- `examples/projects/health/case_check_mocked_not_serving.gctf`
- `examples/projects/health/case_watch_mocked_stream.gctf`
- `examples/projects/health/case_check_gripmock_protected.gctf`
