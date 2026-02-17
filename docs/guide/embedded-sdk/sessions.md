# Session Management <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

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
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession("test-session-123"), // Isolate this test's stubs and history
    )
    require.NoError(t, err)

    // Stubs defined in this session are isolated from other sessions
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "session-test")).
        Reply(sdk.Data("result", "session-isolated")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "session-test"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "session-isolated", resp.Result)
}
```

## Session Isolation Benefits

Sessions provide several benefits:

- **Test Isolation**: Prevents stubs from one test affecting another
- **Parallel Test Safety**: Allows safe parallel execution when sharing a remote GripMock instance
- **History Separation**: Keeps call history separate between different test contexts
- **Resource Management**: Enables cleanup of test-specific resources

## Session Best Practices

### 1. Use Unique Session IDs

Always use unique session identifiers to prevent conflicts:

```go
// Good: Use test name as session ID for uniqueness
mock, err := sdk.Run(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession(t.Name()), // Uses test function name as session ID
)

// Good: Use UUID for guaranteed uniqueness
sessionID := uuid.New().String()
mock, err := sdk.Run(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession(sessionID),
)
```

### 2. Clean Up Sessions

`mock.Close()` cleans remote stubs associated with the active session. You can also set a TTL to trigger automatic cleanup:

```go
func TestMyService_WithCleanup(t *testing.T) {
    sessionID := "test-" + t.Name()
    
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession(sessionID),
        sdk.WithSessionTTL(30 * time.Second),
    )
    require.NoError(t, err)

    // Test logic here...
    
    // Resources for this session are cleaned on Close() and via TTL.
}
```

### 3. Session-Aware Verification

When using sessions, verification occurs within the context of that session:

```go
func TestMyService_SessionVerification(t *testing.T) {
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()),
    )
    require.NoError(t, err)

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "verify-test")).
        Reply(sdk.Data("result", "verified")).
        Times(2). // Expected to be called exactly 2 times in this session
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "verify-test"})

    // ASSERT
    // Verification happens within the session context
    mock.Verify().Method("MyService", "MyMethod").Called(t, 2)
}
```

## Session Configuration

Sessions can be configured with various options depending on your needs:

### Session Timeouts

By default, SDK schedules remote session cleanup with TTL `60s`. Use `sdk.WithSessionTTL(...)` to override:

```go
mock, err := sdk.Run(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession(t.Name()),
    sdk.WithSessionTTL(2*time.Minute),
)
require.NoError(t, err)
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

    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()), // Each parallel test gets its own session
    )
    require.NoError(t, err)

    // Rest of test...
}
```

### Integration Testing Pattern

For integration tests that need shared state, create the mock in test setup code that has access to `t` (for example in suite setup helpers):

```go
func runSharedSessionMock(t *testing.T) sdk.Mock {
    t.Helper()

    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession("integration-suite"),
    )
    require.NoError(t, err)

    return mock
}
```

## Session Limitations

- Sessions are only applicable when using remote mode (`sdk.WithRemote`)
- Session IDs should be unique to prevent conflicts
- Session data persists until explicitly cleared or the server restarts/cleans up
- Each session consumes server resources, so avoid creating excessive numbers of sessions
