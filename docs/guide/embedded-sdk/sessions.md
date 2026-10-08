# Session Management <VersionTag version="v3.16.0" />

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" />. Current API since <VersionTag version="v3.16.0" />; the legacy `sdk.Run` / `mock.Stub` / `mock.Verify` API was **removed in v3.20.0**. See the [Upgrade Guide](./upgrade.md).

Sessions provide isolation for stubs and history data when using remote GripMock instances. Each session maintains its own set of stubs and call history, preventing interference between different test contexts.

## Session Lifecycle

Sessions in GripMock have the following lifecycle characteristics:

1. **Creation**: Sessions are created when the first stub is registered with a specific session ID
2. **Active Period**: During this time, the session stores stubs and history for that session
3. **Automatic Cleanup**: Session resources can be cleaned automatically by the SDK and/or server policies
4. **Manual Cleanup**: Sessions can be explicitly cleared via API calls

## Using Sessions

To use sessions, specify a session ID when connecting to a remote GripMock instance:

```go
func TestMyService_WithSession(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession("test-session-123"), // Isolate this test's stubs and history
    )

    // Stubs defined in this session are isolated from other sessions
    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "session-test").
        Return("result", "session-isolated")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "session-test"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "session-isolated", resp.Result)
}
```

## Why sessions

Stubs and call history are scoped to the session, so several test processes can
share one remote GripMock without seeing each other's stubs or each other's
history. Session-scoped state is also what gets cleaned up when the test ends.

## Choosing a session ID

Two tests sharing an ID share their stubs, which is exactly the failure sessions
exist to prevent. `t.Name()` is unique per test and readable in the UI:

```go
// Good: Use test name as session ID for uniqueness
srv := sdk.NewServer(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession(t.Name()), // Uses test function name as session ID
)

// Good: Use UUID for guaranteed uniqueness
sessionID := uuid.New().String()
srv := sdk.NewServer(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession(sessionID),
)
```

### 2. Clean Up Sessions

`srv.Close()` cleans remote stubs associated with the active session. You can also set an idle TTL as an extra safety net:

```go
func TestMyService_WithCleanup(t *testing.T) {
    sessionID := "test-" + t.Name()
    
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession(sessionID),
        sdk.WithSessionTTL(30 * time.Second),
    )

    // Test logic here...
    
    // Resources for this session are cleaned on Close(), or after 30s of SDK inactivity.
}
```

### 3. Session-Aware Verification

When using sessions, verification occurs within the context of that session:

```go
func TestMyService_SessionVerification(t *testing.T) {
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "verify-test").
        Times(2). // Expected to be called exactly 2 times in this session
        Return("result", "verified")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})

    // ASSERT
    // Verification happens within the session context
    require.Equal(t, 2, srv.Called(MyService_MyMethod_FullMethodName))
}
```

## Session Configuration

Sessions can be configured with various options depending on your needs:

### Session Timeouts

There is no client-side TTL by default: stubs are removed by `srv.Close()` (registered through `t.Cleanup`), and a session abandoned by a crashed test process is reaped by the server itself after `SESSION_GC_TTL`.

`sdk.WithSessionTTL(...)` opts into an extra **idle** timer: it removes the stubs this SDK instance registered only when no SDK operation (stub registration, history read, history purge) happened for the given duration, and it is restarted by every such operation, so it never deletes stubs from under a running test:

```go
srv := sdk.NewServer(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession(t.Name()),
    sdk.WithSessionTTL(2*time.Minute),
)
```

### Session Persistence

Sessions maintain state as long as the remote GripMock instance is running and the session hasn't expired:

- Registered stubs persist within the session
- Call history accumulates within the session
- Verification data is maintained per session

## Common Session Patterns

### Parallel Testing Pattern

When running tests in parallel with a shared remote GripMock instance:

```go
func TestMyService_Parallel(t *testing.T) {
    t.Parallel() // Safe with sessions

    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()), // Each parallel test gets its own session
    )

    // Rest of test...
}
```

### Integration Testing Pattern

For integration tests that need shared state, create the mock in test setup code that has access to `t` (for example in suite setup helpers):

```go
func runSharedSessionMock(t *testing.T) *sdk.Server {
    t.Helper()

    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession("integration-suite"),
    )

    return srv
}
```

## Session Limitations

- Sessions behave the same in both modes; they matter most in remote mode (`sdk.WithRemote`)
- `Session(id)` scopes one stub; `sdk.WithSession(id)` scopes every call the server makes
- Session data persists until it is cleared, its TTL expires, or the server restarts
- Each session holds its own stubs and history on the server
