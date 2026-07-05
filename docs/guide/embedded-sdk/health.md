# Health Checks <VersionTag version="v3.9.3" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

> **Version history:** Health service stubbing available since <VersionTag version="v3.9.3" /> (JSON stubs). Embedded SDK support since <VersionTag version="v3.9.3" /> (legacy API). Current v2 API since <VersionTag version="v3.16.0" />.

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
srv := sdk.NewServer(t, sdk.WithProtoFiles("examples/projects/greeter/service.proto"))
defer srv.Close()

// This stub is stored but always overridden by internal stub:
srv.ExpectUnary("/grpc.health.v1.Health/Check").
    Match("service", "gripmock").
    Return("status", "NOT_SERVING")

client := grpc_health_v1.NewHealthClient(srv.Conn())
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
    srv := sdk.NewServer(t, sdk.WithProtoFiles("examples/projects/greeter/service.proto"))
    defer srv.Close()

    srv.ExpectUnary("/grpc.health.v1.Health/Check").
        Match("service", "examples.health.backend").
        Return("status", "NOT_SERVING")

    client := grpc_health_v1.NewHealthClient(srv.Conn())
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
    srv := sdk.NewServer(t, sdk.WithProtoFiles("examples/projects/greeter/service.proto"))
    defer srv.Close()

    // Even with a stub that targets "gripmock"...
    srv.ExpectUnary("/grpc.health.v1.Health/Check").
        Match("service", "gripmock").
        Return("status", "NOT_SERVING")

    client := grpc_health_v1.NewHealthClient(srv.Conn())
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
    srv := sdk.NewServer(t, sdk.WithProtoFiles("examples/projects/greeter/service.proto"))
    defer srv.Close()

    client := grpc_health_v1.NewHealthClient(srv.Conn())
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
    srv := sdk.NewServer(t, sdk.WithProtoFiles("examples/projects/greeter/service.proto"))
    defer srv.Close()

    srv.ExpectServerStream("/grpc.health.v1.Health/Watch").
        Match("service", "examples.health.watch").
        SendStream(
            map[string]any{"status": "NOT_SERVING"},
            map[string]any{"status": "SERVING"},
        )

    client := grpc_health_v1.NewHealthClient(srv.Conn())
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

You can add a delay before individual stream messages using `Delay`:

```go
srv.ExpectServerStream("/grpc.health.v1.Health/Watch").
    Match("service", "examples.health.watch").
    SendStream(
        Delay(10*time.Millisecond, "status", "NOT_SERVING"),
        map[string]any{"status": "SERVING"},
    )
```

## Full runnable example

See:

- `examples/projects/health/stubs.yaml`
- `examples/projects/health/case_check_mocked_not_serving.gctf`
- `examples/projects/health/case_watch_mocked_stream.gctf`
- `examples/projects/health/case_check_gripmock_protected.gctf`
