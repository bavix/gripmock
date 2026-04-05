# Health Service Stubbing <VersionTag version="v3.9.3" />

GripMock supports stubbing the standard gRPC health service:

- `grpc.health.v1.Health/Check`
- `grpc.health.v1.Health/Watch`

This is useful when you want to test client behavior for dependency health transitions (for example `NOT_SERVING -> SERVING`).

## Protected internal key

The service key `gripmock` is reserved for GripMock internal readiness and cannot be mocked.

- `service: "gripmock"` always returns the real server health.
- Empty service name (`""`) also uses real health behavior.
- Stubs targeting `gripmock` may be stored, but they are ignored at runtime.

## Behavior matrix

| Request `service` | Stub exists | Result |
|---|---|---|
| `gripmock` | Yes/No | Real server health (stub ignored) |
| `""` | Yes/No | Real server health (stub ignored) |
| Custom name | Yes | Mocked response |
| Custom name | No | Default health behavior (`Check` returns `NotFound`) |

## Example: `Check`

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

- service: grpc.health.v1.Health
  method: Check
  input:
    equals:
      service: gripmock
  output:
    data:
      status: NOT_SERVING

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

For `service: "gripmock"`, the saved stub above is intentionally ignored and the real runtime status is returned.

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

```go
mock := mustRunWithProto(t, sdkProtoPath("greeter"))

mock.Stub("grpc.health.v1.Health", "Check").
    When(sdk.Equals("service", "examples.health.backend")).
    Reply(sdk.Data("status", "NOT_SERVING")).
    Commit()

client := grpc_health_v1.NewHealthClient(mock.Conn())
resp, err := client.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{
    Service: "examples.health.backend",
})
require.NoError(t, err)
require.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.GetStatus())
```

## Full runnable example

See:

- `examples/projects/health/stubs.yaml`
- `examples/projects/health/case_check_mocked_not_serving.gctf`
- `examples/projects/health/case_watch_mocked_stream.gctf`
- `examples/projects/health/case_check_gripmock_protected.gctf`
