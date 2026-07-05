# GripMock SDK

Go SDK for embedding a gRPC mock server in tests or connecting to a remote GripMock instance.

> **⚠️ Experimental SDK**  
> This SDK is experimental and may be discontinued or never released. Use at your own risk.

## Quick start (v2 API — recommended)

```go
srv := sdk.NewServer(t, sdk.WithFileDescriptor(helloworld.File_service_proto))
defer srv.Close()

srv.ExpectUnary("/helloworld.Greeter/SayHello").
    Match("name", "Alex").
    Return("message", "Hi Alex")

client := helloworld.NewGreeterClient(srv.Conn())
resp, _ := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Alex"})
// resp.Message == "Hi Alex"
```

`NewServer` creates an independent server (safe for `t.Parallel()`), starts it on a random TCP port, and registers `t.Cleanup` for auto-verify + close.

**Unary** with delay:
```go
srv.ExpectUnary("/svc/Method").
    Match("field", "value").
    Return(Delay(100*time.Millisecond, "responseField", "responseValue"))
```

**Unary** with sequential responses (`NextWillReturn`):
```go
srv.ExpectUnary("/svc/Method").
    Match("step", "first").
    Return("result", "ok").
    NextWillReturn("result", "again")
// 1st call → {result: ok}, 2nd call → {result: again}, 3rd call → error
```

**Server Stream**:
```go
srv.ExpectServerStream("/svc/Stream").
    Match("query", "test").
    SendStream(
        map[string]any{"id": "1", "title": "result 1"},
        map[string]any{"id": "2", "title": "result 2"},
    )
```

**Client Stream**:
```go
srv.ExpectClientStream("/svc/Stream").
    Match(sdk.Matches("value", "\\d+")).
    Return("result", 42.0)
```

**Bidirectional Stream**:
```go
srv.ExpectBidirectionalStream("/svc/Bidi").
    Run(func(ctx context.Context, stream any) error {
        return nil
    })
```

**Effects** (register side-effect stubs on match):
```go
effect := sdk.Upsert("svc", "NextMethod").
    Match("step", "complete").
    Return("status", "done").
    Build()

srv.ExpectUnary("/svc/Method").
    Match("step", "begin").
    Effect(effect).
    Return("status", "started")
```

**Verification**:
```go
err := srv.ExpectationsWereMet()
n := srv.Called("/svc/Method")
total := srv.TotalCalls()
history := srv.History()
```

**Matchers** for fine-grained matching:
```go
// Equals (default), Contains, Matches (regex), Glob
srv.ExpectUnary("/svc/Method").
    Match("name", sdk.Contains("partial")).
    Match(sdk.Matches("email", `.*@example\.com$`)).
    Match(sdk.Glob("path", "prefix/**/suffix")).
    Match("exact_field", "exact_value").
    Return("result", "matched")

// Header matching — same universal matchers, pass to WithHeader/WithHeaders
srv.ExpectUnary("/svc/Method").
    WithHeader(sdk.Contains("authorization", "Bearer ")).
    Return("result", "auth_ok")

// AnyOf, And, IgnoreArrayOrder for complex conditions
srv.ExpectUnary("/svc/Method").
    Match(sdk.AnyOf(
        sdk.Equals("status", "active"),
        sdk.Equals("status", "pending"),
    )).
    Return("result", "ok")
```

### Advanced features

**Priority** (higher wins):
```go
srv.ExpectUnary("/svc/Method").
    Match("id", "specific").
    Priority(100).Return("result", "specific")

srv.ExpectUnary("/svc/Method").
    Match(sdk.Contains("id", "")).
    Priority(10).Return("result", "generic")
```

**Times** (call limit):
```go
srv.ExpectUnary("/svc/Method").
    Match("id", "limited").
    Times(3).Return("result", "ok")
// Only matches 3 times, 4th call returns error
```

**ReturnError**:
```go
srv.ExpectUnary("/svc/Method").
    Match("amount", 0).
    ReturnError(codes.InvalidArgument, "amount must be positive")
```

---

## Migration guide: v1 → v2

### Key differences

| v1 (deprecated) | v2 (recommended) |
|---|---|
| `sdk.Run(t, opts...)` | `sdk.NewServer(t, opts...)` |
| `mock.Stub(svc, method).Unary(...).Commit()` | `srv.ExpectUnary(fullMethod).Match(...).Return(...)` |
| `sdk.By(fullMethod)` | use full method string directly |
| `mock.Verify().Method(...).Called(t, n)` | `srv.Called(fullMethod)` |
| `mock.History()` | `srv.History()` |
| `mock.Stub(...).Delay(d).Commit()` | `Return(Delay(d, ...))` or `SendStream(Delay(d, ...))` |
| `mock.Stub(...).WhenStream(...)` | `ExpectClientStream(...).Match(...)` |
| `mock.Stub(...).ReplyStream(...)` | `ExpectServerStream(...).SendStream(...)` |

### Removed in v2

- `sdk.By()`, `sdk.ParseFullMethodName()`, `sdk.StubBatch()` — use full method strings directly
- `MockFrom()` → `WithReflection()`
- `Remote()` → `WithRemote()` (auto-derives REST URL from gRPC address)
- `WithServiceDesc()` — no-op, removed
- `WithPayloadFunc()` — no-op, removed
- `bufconn` — default is now TCP `:0` (real address like `127.0.0.1:PORT`)
- `MergeHeaders()`, `MergeOutput()` — replaced by Matcher-based API

### New in v2

- 4 dedicated expectation types: `UnaryExpectation`, `ServerStreamExpectation`, `ClientStreamExpectation`, `BidirectionalExpectation`
- `NextWillReturn(kv...)` — sequential responses
- `Delay(d, kv...)` — composable delay inside `Return`
- `Effect(Upsert(...).Build())` — side effects on match
- `WithBatch()` — optional batching for remote mode
- `Server.Reset()` — clear state without stopping server
- `Server.Flush()` — flush pending stubs (batch mode)
- `Server.Address()` — get listen address
- Unified `Matcher` type: `Equals`, `Contains`, `Matches`, `Glob`, `AnyOf`, `And` for payload AND header matching (pass to `Match()` for payload, `WithHeader()` for headers)

---

## Options

| Option | Description |
|--------|-------------|
| `WithFileDescriptor(fd)` | Use generated descriptor (e.g. `helloworld.File_service_proto`) |
| `WithDescriptors(fds)` | Use `FileDescriptorSet` (one or more files); can be chained |
| `WithProtoFiles(paths...)` | Compile `.proto` files at test time |
| `WithListenAddr(network, addr)` | Listen on a specific address (default: `:0`) |
| `WithRemote(grpcAddr, restURL)` | Connect to an external GripMock |
| `WithSession(id)` | Session isolation for parallel tests |
| `WithSessionTTL(d)` | Cleanup window for session resources |
| `WithGRPCTimeout(d)` | Per-RPC timeout for remote gRPC calls |
| `WithHealthCheckTimeout(d)` | Readiness check timeout |
| `WithReflection(addr)` | Load descriptors via gRPC reflection |
| `WithHTTPClient(c)` | Custom HTTP client for remote mode |
| `WithBatch()` | Enable stub batching for remote mode (flush via `Flush()`) |

## Remote mode

```go
srv := sdk.NewServer(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession("suite-A"),
)
defer srv.Close()

srv.ExpectUnary("/helloworld.Greeter/SayHello").
    Match("name", "Alex").
    Return("message", "Hi Alex")

client := helloworld.NewGreeterClient(srv.Conn())
```

Use `WithBatch()` to queue stubs and send them all at once via `Flush()`:
```go
srv := sdk.NewServer(t, sdk.WithRemote("localhost:4770", "http://localhost:4771"), sdk.WithBatch())
defer srv.Close()

srv.ExpectUnary("/svc/M").Match("k", "v").Return("r", "1")
srv.ExpectUnary("/svc/M").Match("k2", "v2").Return("r", "2")
srv.Flush() // sends both stubs in one REST call
```

## Lifecycle methods

```go
srv := sdk.NewServer(t, opts...)
srv.Address()               // "127.0.0.1:PORT"
srv.Conn()                  // *grpc.ClientConn
srv.Reset()                 // clear state without stopping
srv.Flush()                 // flush pending stubs (batch mode)
srv.Called("/svc/Method")   // call count
srv.TotalCalls()            // total call count
srv.History()               // all recorded calls
srv.ExpectationsWereMet()   // verify Times expectations
srv.Close()                 // stop + cleanup
```

## Installation

```bash
go get github.com/bavix/gripmock/v3/pkg/sdk
```
