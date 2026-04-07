# Health Service Stubbing <VersionTag version="v3.9.3" />

GripMock supports stubbing the standard gRPC health service:

- `grpc.health.v1.Health/Check`
- `grpc.health.v1.Health/Watch`

This is useful when you want to test client behavior for dependency health transitions (for example `NOT_SERVING -> SERVING`).

## Protected service: gripmock

The service key `gripmock` is reserved for GripMock internal readiness.

- Internal stub with `service: "gripmock"` is created automatically at server startup with `NOT_SERVING` status
- When server becomes ready, status updates to `SERVING`
- User stubs targeting `service: "gripmock"` are stored but always overridden by internal stub (priority)
- Do not rely on mocking `gripmock` — it will not work as expected

## Behavior matrix

| Request `service` | Internal stub | User stub | Result |
|---|---|---|---|
| `gripmock` | Yes (automatic) | Yes/No | Internal stub → SERVING |
| Custom name | No | Yes | User stub response |
| Custom name | No | No | Default behavior (`Check` returns `NotFound`) |
| Empty `""` | No | Yes | Mocked response |
| Empty `""` | No | No | Default behavior |

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
