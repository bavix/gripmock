# Remote Mode <VersionTag version="v3.16.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Remote mode introduced in <VersionTag version="v3.16.0" />. Not available in the legacy API.

Connect to a remote GripMock instance instead of running embedded. When using remote mode, you must provide both the gRPC endpoint (for mock server) and HTTP endpoint (for management operations).

Use `sdk.WithRemote(grpcAddr, restURL)` for remote mode.

::: warning
Remote mode works without `sdk.WithSession(...)`, but this is not recommended for tests.
Without sessions, stubs and history can leak between tests and cause flaky behavior.
:::

## Connecting to Remote GripMock

When connecting to a remote GripMock instance, you must specify both the gRPC endpoint (for the mock server) and the HTTP endpoint (for management operations):

```go
func TestMyService_Remote(t *testing.T) {
    // ARRANGE
    // Connect to a remote GripMock server - specify both gRPC and HTTP endpoints
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),  // gRPC endpoint, HTTP management endpoint
        sdk.WithFileDescriptor(service.File_service_proto),
    )

    // Define stubs in the Arrange phase
    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "remote-test").
        Return("result", "from-remote")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "remote-test"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "from-remote", resp.Result)
}
```

## Session Isolation

```go
func TestMyService_SessionIsolation(t *testing.T) {
    t.Parallel() // Safe with sessions

    // ARRANGE
    // Use a unique session for this test
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()), // Use test name as session ID
    )

    // Stubs in this session are isolated from other tests
    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "isolated").
        Return("result", "isolated_result")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "isolated"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "isolated_result", resp.Result)
}
```

## Health Timeout Configuration

```go
func TestMyService_HealthTimeout(t *testing.T) {
    // ARRANGE
    // Configure the timeout for waiting for the remote server to become healthy
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithHealthCheckTimeout(15 * time.Second), // Wait up to 15 seconds
    )

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "timeout-test").
        Return("result", "success")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "timeout-test"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "success", resp.Result)
}
```

## Remote Mode with Error Handling

```go
func TestMyService_RemoteWithError(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
    )

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "error-case").
        ReturnError(codes.Internal, "Remote service error")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    _, err := client.MyMethod(t.Context(), &MyRequest{Id: "error-case"})

    // ASSERT
    require.Error(t, err)
    require.Equal(t, codes.Internal, status.Code(err))
    require.Contains(t, err.Error(), "Remote service error")
}
```

## Parallel Tests with Remote Sessions

```go
func TestMyService_ParallelExecution(t *testing.T) {
    t.Parallel()

    // ARRANGE
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "parallel-test").
        Return("result", "parallel-success")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "parallel-test"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "parallel-success", resp.Result)
}
```

## Remote Mode with Verification

```go
func TestMyService_RemoteVerification(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "verify-test").
        Times(2). // Expect exactly 2 calls
        Return("result", "verified")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})

    // ASSERT
    // Verification happens automatically due to Times(2) and passing t to NewServer
    require.Equal(t, 2, srv.Called(MyService_MyMethod_FullMethodName))
}
```

## Custom HTTP Client

Use `sdk.WithHTTPClient(...)` when you need custom transport, tracing, or timeouts for REST management calls:

```go
func TestMyService_RemoteWithCustomHTTPClient(t *testing.T) {
    // ARRANGE
    httpClient := &http.Client{Timeout: 3 * time.Second}

    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithHTTPClient(httpClient),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "custom-http").
        Return("result", "ok")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "custom-http"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "ok", resp.Result)
}
```

## Context Propagation for Management Calls

Remote mode uses HTTP management APIs (`/api/stubs`, `/api/history`, `/api/verify`, `/api/descriptors`).

- Verification methods that take `t` use `t.Context()`.
- History/verification can also be called with explicit context helpers:
  - `sdk.HistoryAllContext(...)`
  - `sdk.HistoryCountContext(...)`
  - `sdk.HistoryFilterByMethodContext(...)`
  - `sdk.VerifyStubTimesErrContext(...)`

Use `ExpectationsWereMetContext(ctx)` for context-aware verification:

```go
func TestMyService_RemoteContextCancel(t *testing.T) {
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession(t.Name()),
    )

    // ... Arrange/Act ...

    ctx, cancel := context.WithCancel(t.Context())
    cancel()

    err := srv.ExpectationsWereMetContext(ctx)
    require.Error(t, err)
    require.ErrorIs(t, err, context.Canceled)
}
```

## gRPC Timeout for Remote Calls

Use `sdk.WithGRPCTimeout(...)` to apply a default timeout to remote gRPC calls when request context has no deadline:

```go
func TestMyService_RemoteWithGRPCTimeout(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession(t.Name()),
        sdk.WithGRPCTimeout(250*time.Millisecond),
        sdk.WithFileDescriptor(service.File_service_proto),
    )

    srv.ExpectUnary(MyService_SlowMethod_FullMethodName).
        Return(Delay(2*time.Second, "result", "ok"))

    client := NewMyServiceClient(srv.Conn())

    // ACT
    _, err := client.SlowMethod(t.Context(), &MyRequest{})

    // ASSERT
    require.Error(t, err)
    require.Equal(t, codes.DeadlineExceeded, status.Code(err))
}
```

## Advantages of Remote Mode

- Share a single GripMock instance across multiple test processes
- Better resource utilization in CI environments
- Persistent state between test runs (if needed)
- Ability to inspect state via the web UI

## Disadvantages of Remote Mode

- Network overhead
- Potential for test interference without proper session isolation
- Dependency on external process
- Slower startup compared to embedded mode

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::
