# Health Checks <VersionTag version="v3.9.3" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

GripMock supports stubbing the standard gRPC health service:

- `grpc.health.v1.Health/Check` — unary health status
- `grpc.health.v1.Health/Watch` — streaming health status (useful for testing health transitions)

This allows you to test client behavior for dependency health transitions (for example `NOT_SERVING -> SERVING`).

## Protected service: gripmock

The service key `gripmock` is reserved for GripMock internal readiness.

- An internal stub is created automatically at server startup with `NOT_SERVING` status
- When the server becomes ready, the status updates to `SERVING`
- User stubs targeting `service: "gripmock"` are stored but always overridden by the internal stub

```go
mock := mustRunWithProto(t, sdkProtoPath("greeter"))

// This stub is stored but always overridden by internal stub:
mock.Stub("grpc.health.v1.Health", "Check").
    When(sdk.Equals("service", "gripmock")).
    Reply(sdk.Data("status", "NOT_SERVING")).
    Commit()

client := grpc_health_v1.NewHealthClient(mock.Conn())
resp, err := client.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{
    Service: "gripmock",
})
// Returns SERVING (internal stub)
require.NoError(t, err)
require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.GetStatus())
```

## Mocking Check

```go
func TestHealthCheckMockedViaSDK(t *testing.T) {
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
}
```

## gripmock service behavior

Requests for the `gripmock` service return the internal stub status:

```go
func TestHealthCheckGripmockProtectedViaSDK(t *testing.T) {
    mock := mustRunWithProto(t, sdkProtoPath("greeter"))

    // Even with a stub that targets "gripmock"...
    mock.Stub("grpc.health.v1.Health", "Check").
        When(sdk.Equals("service", "gripmock")).
        Reply(sdk.Data("status", "NOT_SERVING")).
        Commit()

    client := grpc_health_v1.NewHealthClient(mock.Conn())
    resp, err := client.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{
        Service: "gripmock",
    })
    require.NoError(t, err)
    // Internal stub overrides user stub — SERVING
    require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.GetStatus())
}
```

## Unknown service fallback

If no stub matches and the service is not `gripmock`, the request returns `NotFound`:

```go
func TestHealthCheckUnknownServiceFallbackViaSDK(t *testing.T) {
    mock := mustRunWithProto(t, sdkProtoPath("greeter"))

    client := grpc_health_v1.NewHealthClient(mock.Conn())
    resp, err := client.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{
        Service: "examples.health.unknown",
    })
    require.Error(t, err)
    require.Equal(t, codes.NotFound, status.Code(err))
}
```

## Mocking Watch stream

The `Watch` method returns a stream of health status updates. You can stub it to return a sequence of statuses:

```go
func TestRunHealthWatchMockedStreamViaSDK(t *testing.T) {
    mock := mustRunWithProto(t, sdkProtoPath("greeter"))

    mock.Stub("grpc.health.v1.Health", "Watch").
        When(sdk.Equals("service", "examples.health.watch")).
        ReplyStream(
            sdk.Data("status", "NOT_SERVING"),
            sdk.Data("status", "SERVING"),
        ).
        Commit()

    client := grpc_health_v1.NewHealthClient(mock.Conn())
    ctx, cancel := context.WithCancel(t.Context())
    defer cancel()

    stream, err := client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{
        Service: "examples.health.watch",
    })
    require.NoError(t, err)

    first, err := stream.Recv()
    require.NoError(t, err)
    require.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, first.GetStatus())

    second, err := stream.Recv()
    require.NoError(t, err)
    require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, second.GetStatus())
}
```

## Watch with delay

You can add a delay before the stream starts:

```go
mock.Stub("grpc.health.v1.Health", "Watch").
    When(sdk.Equals("service", "examples.health.watch")).
    Reply(sdk.Data("status", "NOT_SERVING"), sdk.Data("status", "SERVING")).
    Delay(10 * time.Millisecond).
    Commit()
```

## Full runnable example

See:

- `examples/projects/health/stubs.yaml`
- `examples/projects/health/case_check_mocked_not_serving.gctf`
- `examples/projects/health/case_watch_mocked_stream.gctf`
- `examples/projects/health/case_check_gripmock_protected.gctf`
