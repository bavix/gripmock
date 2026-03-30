# GripMock SDK

Go SDK for embedding a gRPC mock server in tests or connecting to a remote GripMock instance.

> **⚠️ Experimental SDK**  
> This SDK is experimental and may be discontinued or never released. Use at your own risk.

## Quick start (tests)

```go
mock, err := sdk.Run(t, sdk.WithFileDescriptor(helloworld.File_service_proto))
require.NoError(t, err)

mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()

client := helloworld.NewGreeterClient(mock.Conn())
reply, _ := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Alex"})
// reply.Message == "Hi Alex"
```

`Run` requires a non-nil `TestingT` (e.g. `*testing.T`) and always registers cleanup (`VerifyStubTimesErr` + `Close`).

## Stubbing styles

SDK supports two forms:

- `mock.Stub(service, method)` (backward-compatible)
- `mock.Stub(sdk.By(fullMethod))` (preferred)

Use generated full-method constants from `*_grpc.pb.go` when available:

```go
mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()

mock.Verify().Method(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).Called(t, 1)
```
`sdk.By(...)` accepts `/package.Service/Method` (leading slash optional).

**Unary** — one-liner: match one field, return one field.

```go
mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    Unary("name", "Bob", "message", "Hello Bob").
    Commit()
```

**Match + Return** — key-value pairs for input and output.

```go
mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    Match("name", "Bob").
    Return("message", "Hello Bob").
    Commit()
```

**Dynamic template** — interpolate request fields into response.

```go
mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    When(sdk.Matches("name", ".+")).
    Return("message", "Hi {{.Request.name}}").
    Commit()
// Request: name="Alex" → Response: message="Hi Alex"
```

**Delay** — simulate slow responses before sending the reply.

```go
mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    Unary("name", "Bob", "message", "Hello Bob").
    Delay(100 * time.Millisecond).
    Commit()
// Response is sent after 100ms delay
```

## Verification

When stubs use `Times`, `Run(t, ...)` verifies call count at test end automatically.

For explicit checks, use `mock.Verify()` and `mock.History()`.

In remote mode, management REST calls are context-aware:

- `mock.Verify().Method(sdk.By(...)).Called(t, n)`, `mock.Verify().Total(t, n)`, and `mock.Verify().VerifyStubTimes(t)` use `t.Context()`.
- You can pass explicit context with helper functions:

```go
ctx := context.WithValue(t.Context(), traceKey{}, "suite-A")

err := sdk.VerifyStubTimesErrContext(ctx, mock.Verify())
require.NoError(t, err)

calls, err := sdk.HistoryAllContext(ctx, mock.History())
require.NoError(t, err)
require.NotEmpty(t, calls)

count, err := sdk.HistoryCountContext(ctx, mock.History())
require.NoError(t, err)
require.GreaterOrEqual(t, count, 1)

filtered, err := sdk.HistoryFilterByMethodContext(ctx, mock.History(), "helloworld.Greeter", "SayHello")
require.NoError(t, err)
require.NotEmpty(t, filtered)
```

These helpers are backward-compatible:

- For remote verifier/history they use the provided context.
- For embedded verifier/history they gracefully fall back to non-context methods.

## Multiple services (one mock)

One mock can serve N services. Pass several descriptors via chained options (duplicates by file name are skipped):

```go
// Option 1: multiple WithFileDescriptor (generated code)
mock, err := sdk.Run(t,
    sdk.WithFileDescriptor(helloworld.File_service_proto),
    sdk.WithFileDescriptor(echo.File_service_v1_proto),
)
require.NoError(t, err)

mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()
mock.Stub(sdk.By(echo.EchoService_SendMessage_FullMethodName)).
    Unary("message", "ping", "response", "pong").
    Commit()
```

```go
// Option 2: multiple WithDescriptors (FileDescriptorSet from protoc, buf, etc.)
mock, err := sdk.Run(t,
    sdk.WithDescriptors(fdsGreeter),
    sdk.WithDescriptors(fdsEcho),
)
require.NoError(t, err)
```

```go
// Option 3: single WithDescriptors with merged FileDescriptorSet
fds := &descriptorpb.FileDescriptorSet{
    File: append(fdsGreeter.GetFile(), fdsEcho.GetFile()...),
}
mock, err := sdk.Run(t, sdk.WithDescriptors(fds))
```

## Run options

| Option | Description |
|--------|-------------|
| `WithFileDescriptor(fd)` | Use generated descriptor (e.g. `helloworld.File_service_proto`) |
| `WithDescriptors(fds)` | Use `FileDescriptorSet` (one or more files); can be chained for multiple protos |
| `WithListenAddr(network, addr)` | Listen on a real port (e.g. `"tcp", ":0"`) |
| `WithRemote(grpcAddr, restURL)` | Connect to an external GripMock (gRPC + REST) |
| `Remote(grpcAddr)` | Deprecated alias; derives REST URL automatically |
| `WithSession(id)` | Session isolation for parallel tests (remote only) |
| `WithSessionTTL(d)` | Automatic cleanup window for session resources (remote only) |
| `WithGRPCTimeout(d)` | Default per-RPC timeout for remote gRPC calls |

## Remote mode

`WithFileDescriptor(...)` / `WithDescriptors(...)` are optional in remote mode.
When provided in remote mode, SDK uploads descriptors to `/api/descriptors` on startup.
Descriptors remain required for embedded mode.

`WithHTTPClient(...)` customizes the REST client used for remote management calls.

```go
mock, err := sdk.Run(t,
    sdk.WithRemote("localhost:4770", "http://localhost:4771"),
    sdk.WithSession("suite-A"),
)
require.NoError(t, err)

mock.Stub(sdk.By(helloworld.Greeter_SayHello_FullMethodName)).
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()

client := helloworld.NewGreeterClient(mock.Conn())
```

## Installation

```bash
go get github.com/bavix/gripmock/v3/pkg/sdk
```
