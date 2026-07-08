# Upgrading to v3.16.0 <VersionTag version="v3.16.0" />

::: warning
⚠️ The old stub-definition API (`sdk.Run`, `mock.Stub`, `Mock.Verify`, etc.) is deprecated but still available. All new code should use the new API.
:::

## Why the change?

The original API grew organically — one generic `mock.Stub()` for every gRPC pattern, `.Commit()` to finish every chain, `sdk.By()` to reference methods, and verification buried under `mock.Verify().Method(...)`.

It worked, but it had friction:

- `mock.Stub()` + `.When()` + `.Reply()` treated unary, streaming, and bidi the same — the reader couldn't tell which pattern was being tested without scanning the whole chain
- `Commit()` was easy to forget, silently breaking expectations
- `.When(sdk.Equals("k", "v"))` was verbose — Equals is the default, so `Match("k", "v")` says the same thing in half the characters
- `.Reply(sdk.Data("k", "v"))` → `.Return("k", "v")` — less nesting, less noise
- `sdk.By()` was an unnecessary wrapper; full method strings are unambiguous
- Verification required a detour through `mock.Verify()`, adding ceremony to a simple assertion
- `sdk.Run()` returned an error that was always checked with `require.NoError` — noise, never a real error in practice

**v3.16.0** ditches the generic approach. Every gRPC interaction pattern gets its own expectation type, verification moves to direct server methods, and `Commit()` disappears entirely.

## Quick Reference

### Server creation

```go
// old
mock, err := sdk.Run(t, sdk.WithFileDescriptor(proto))
require.NoError(t, err)
defer mock.Close()

// new
srv := sdk.NewServer(t, sdk.WithFileDescriptor(proto))
defer srv.Close()
```

### Unary

```go
// old
mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    When(sdk.Equals("name", "Alex")).
    Reply(sdk.Data("message", "Hi Alex")).
    Commit()

// new
srv.ExpectUnary(helloworld.Greeter_SayHello_FullMethodName).
    Match("name", "Alex").
    Return("message", "Hi Alex")
```

### Server-stream

```go
// old
mock.Stub(sdk.By(FullMethod)).
    When(sdk.Equals("query", "test")).
    ReplyStream(
        sdk.Data("id", "1"),
        sdk.Data("id", "2"),
    ).
    Commit()

// new
srv.ExpectServerStream(FullMethod).
    Match("query", "test").
    SendStream(
        map[string]any{"id": "1"},
        map[string]any{"id": "2"},
    )
```

### Client-stream

```go
// old
mock.Stub(sdk.By(FullMethod)).
    WhenStream("value", "\\d+").
    Reply(sdk.Data("result", 42.0)).
    Commit()

// new
srv.ExpectClientStream(FullMethod).
    Match(sdk.Matches("value", "\\d+")).
    Return("result", 42.0)
```

### Bidirectional

```go
// old
mock.Stub(sdk.By(FullMethod)).ReplyStream().Commit()

// new
srv.ExpectBidirectionalStream(FullMethod).
    Run(func(ctx context.Context, stream any) error {
        return nil
    })
```

## Mapping Table

| Old | New |
|-----|-----|
| `sdk.Run(t, opts...)` | `sdk.NewServer(t, opts...)` |
| `mock.Stub(svc, method)` | `srv.ExpectUnary("/svc/Method")` |
| `mock.Stub(sdk.By(fullMethod))` | `srv.ExpectUnary(fullMethod)` |
| `.When(sdk.Equals("k", "v"))` | `Match("k", "v")` |
| `.When(sdk.Contains("k", "v"))` | `Match(sdk.Contains("k", "v"))` |
| `.When(sdk.Matches("k", "re"))` | `Match(sdk.Matches("k", "re"))` |
| `.WhenHeaders(sdk.Equals("k", "v"))` | `WithHeader(sdk.Equals("k", "v"))` |
| `.Reply(sdk.Data("k", "v"))` | `Return("k", "v")` |
| `.ReplyError(code, msg)` | `ReturnError(code, msg)` |
| `.ReplyStream(msg...)` | `SendStream(map...)` (on `ExpectServerStream`) |
| `.Delay(d).Commit()` | `Return(Delay(d, kv...))` |
| `.Times(n).Commit()` | `Times(n).Return(...)` |
| `.Priority(n).Commit()` | `Priority(n).Return(...)` |
| `.Commit()` | **removed** — auto-registered |
| `mock.Verify().Method(fm).Called(t, n)` | `require.Equal(t, n, srv.Called(fm))` |
| `mock.Verify().Total(t, n)` | `require.Equal(t, n, srv.TotalCalls())` |
| `mock.Verify().VerifyStubTimes(t)` | `srv.ExpectationsWereMet()` |
| `mock.History().FilterByMethod(...)` | `srv.History()` |
| `mock.Conn()` | `srv.Conn()` |
| `mock.Close()` | `srv.Close()` |
| `sdk.By(fullMethod)` | use full method string directly |
| `sdk.StubBatch()` | `NextWillReturn(kv...)` |

## By Example

### Matching

```go
// old — verbose, even for default Equals
.When(sdk.Equals("name", "Alex"))
.When(sdk.Contains("name", "partial"))
.WhenHeaders(sdk.Equals("authorization", "Bearer token"))

// new — Equals is the default for payload, WithHeader for headers
.Match("name", "Alex")
.Match(sdk.Contains("name", "partial"))
.WithHeader(sdk.Equals("authorization", "Bearer token"))
```

### Delays

```go
// old — delay was a separate modifier before Commit
mock.Stub(sdk.By(FullMethod)).
    When(sdk.Equals("id", "slow")).
    Reply(sdk.Data("result", "ok")).
    Delay(100 * time.Millisecond).
    Commit()

// new — delay is part of the return value
srv.ExpectUnary(FullMethod).
    Match("id", "slow").
    Return(Delay(100*time.Millisecond, "result", "ok"))
```

### Sequential responses

```go
// old — no clean way, needed StubBatch or multiple stubs

// new — NextWillReturn chains additional responses
srv.ExpectUnary(FullMethod).
    Match("id", "seq").
    Return("step", "first").
    NextWillReturn("step", "second")

// NextWillReturnError for error sequences
srv.ExpectUnary(FullMethod).
    Match("id", "retry").
    ReturnError(codes.Unavailable, "try again").
    NextWillReturnError(codes.Unavailable, "one more").
    NextWillReturn("result", "success")
```

### Effects

```go
// new — side-effect stubs, no equivalent in old API
effect := sdk.Upsert("svc", "NextMethod").
    Match("step", "complete").
    Return("status", "done").
    Build()

srv.ExpectUnary("/svc/Method").
    Match("step", "begin").
    Effect(effect).
    Return("status", "started")

```

### Verification

```go
// old — detour through mock.Verify()
mock.Verify().Method(sdk.By(FullMethod)).Called(t, 2)
mock.Verify().Total(t, 5)
mock.Verify().Method(sdk.By(FullMethod)).Never(t)
mock.History().FilterByMethod("svc", "method")

// new — direct server methods
require.Equal(t, 2, srv.Called(FullMethod))
require.Equal(t, 5, srv.TotalCalls())
require.Equal(t, 0, srv.Called(FullMethod))
calls := srv.History()
```

## What Was Removed

- `sdk.By()` — not needed, full method strings work directly
- `Commit()` — auto-registered on terminal call
- `sdk.StubBatch()` — replaced by `NextWillReturn`
- `sdk.MergeHeaders()` / `sdk.MergeOutput()` — replaced by unified `Match()` API
- `WithPayloadFunc()` — was a no-op placeholder
- `mock.Verify()` — replaced by `srv.Called()`, `srv.TotalCalls()`, `srv.History()`

## What's New

- `ExpectUnary(fullMethod)` — dedicated expectations for each gRPC pattern
- `ExpectServerStream(fullMethod)` + `SendStream()`
- `ExpectClientStream(fullMethod)` + `Match()`
- `ExpectBidirectionalStream(fullMethod)` + `Run(fn)`
- `NextWillReturn(kv...)` — sequential responses on any expectation
- `Return(Delay(d, kv...))` — composable delay inside the return value
- `Effect(Upsert(...).Build())` — side-effect stubs
- `Server.Reset()` / `Server.Flush()` / `Server.Address()` — more lifecycle methods
- `WithBatch()` — batch mode for remote
- Unified `Matcher` types: `Equals`, `Contains`, `Matches`, `Glob`, `AnyOf`, `And`, `IgnoreArrayOrder`

## Old API Still Available

The old API (`sdk.Run`, `mock.Stub`, `mock.Verify`, etc.) still works via `v1compat.go`. Both APIs can coexist in the same test suite — migrate at your own pace.

```go
// old API — still works
mock, err := sdk.Run(t, sdk.WithFileDescriptor(helloworld.File_service_proto))
mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    When(sdk.Equals("name", "Alex")).
    Reply(sdk.Data("message", "Hi Alex")).
    Commit()
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::
