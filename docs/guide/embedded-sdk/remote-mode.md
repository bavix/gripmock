# Remote Mode <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

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
    mock, err := sdk.Run(t, 
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),  // gRPC endpoint, HTTP management endpoint
        sdk.WithFileDescriptor(service.File_service_proto),
    )
    require.NoError(t, err)

    // Define stubs in the Arrange phase
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "remote-test")).
        Reply(sdk.Data("result", "from-remote")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()), // Use test name as session ID
    )
    require.NoError(t, err)

    // Stubs in this session are isolated from other tests
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "isolated")).
        Reply(sdk.Data("result", "isolated_result")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithHealthCheckTimeout(15 * time.Second), // Wait up to 15 seconds
    )
    require.NoError(t, err)

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "timeout-test")).
        Reply(sdk.Data("result", "success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
    )
    require.NoError(t, err)

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "error-case")).
        ReplyError(codes.Internal, "Remote service error").
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    _, err = client.MyMethod(t.Context(), &MyRequest{Id: "error-case"})

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
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )
    require.NoError(t, err)

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "parallel-test")).
        Reply(sdk.Data("result", "parallel-success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )
    require.NoError(t, err)

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "verify-test")).
        Reply(sdk.Data("result", "verified")).
        Times(2). // Expect exactly 2 calls
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})

    // ASSERT
    // Verification happens automatically due to Times(2) and passing t to Run
    mock.Verify().Method("MyService", "MyMethod").Called(t, 2)
}
```

## Custom HTTP Client

Use `sdk.WithHTTPClient(...)` when you need custom transport, tracing, or timeouts for REST management calls:

```go
func TestMyService_RemoteWithCustomHTTPClient(t *testing.T) {
    // ARRANGE
    httpClient := &http.Client{Timeout: 3 * time.Second}

    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithHTTPClient(httpClient),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )
    require.NoError(t, err)

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "custom-http")).
        Reply(sdk.Data("result", "ok")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "custom-http"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "ok", resp.Result)
}
```

## gRPC Timeout for Remote Calls

Use `sdk.WithGRPCTimeout(...)` to apply a default timeout to remote gRPC calls when request context has no deadline:

```go
func TestMyService_RemoteWithGRPCTimeout(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession(t.Name()),
        sdk.WithGRPCTimeout(250*time.Millisecond),
        sdk.WithFileDescriptor(service.File_service_proto),
    )
    require.NoError(t, err)

    mock.Stub("MyService", "SlowMethod").
        Reply(sdk.Data("result", "ok")).
        Delay(2 * time.Second).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    _, err = client.SlowMethod(t.Context(), &MyRequest{})

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
