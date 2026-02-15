# GripMock SDK

Go SDK for embedding a gRPC mock server in tests or connecting to a remote GripMock instance.

> **⚠️ Experimental SDK**  
> This SDK is experimental and may be discontinued or never released. Use at your own risk.

## Quick start (tests)

```go
mock, err := sdk.Run(t, sdk.WithFileDescriptor(helloworld.File_service_proto))
require.NoError(t, err)

mock.Stub("helloworld.Greeter", "SayHello").
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()

client := helloworld.NewGreeterClient(mock.Conn())
reply, _ := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Alex"})
// reply.Message == "Hi Alex"
```

`Run` registers cleanup (verify stub `Times` + `Close`) when `t` is non-nil — no defer needed.

## Stubbing styles

**Unary** — one-liner: match one field, return one field.

```go
mock.Stub("helloworld.Greeter", "SayHello").
    Unary("name", "Bob", "message", "Hello Bob").
    Commit()
```

**Match + Return** — key-value pairs for input and output.

```go
mock.Stub("helloworld.Greeter", "SayHello").
    Match("name", "Bob").
    Return("message", "Hello Bob").
    Commit()
```

**Dynamic template** — interpolate request fields into response.

```go
mock.Stub("helloworld.Greeter", "SayHello").
    When(sdk.Matches("name", ".+")).
    Return("message", "Hi {{.Request.name}}").
    Commit()
// Request: name="Alex" → Response: message="Hi Alex"
```

**Delay** — simulate slow responses before sending the reply.

```go
mock.Stub("helloworld.Greeter", "SayHello").
    Unary("name", "Bob", "message", "Hello Bob").
    Delay(100 * time.Millisecond).
    Commit()
// Response is sent after 100ms delay
```

## Verification

When stubs use `Times`, pass `t` to `Run` — it verifies call count at test end. For `Run(nil, ...)`, call `mock.Verify().VerifyStubTimesErr()` before `Close()`.

## Multiple services (one mock)

One mock can serve N services. Pass several descriptors via chained options (duplicates by file name are skipped):

```go
// Option 1: multiple WithFileDescriptor (generated code)
mock, err := sdk.Run(t,
    sdk.WithFileDescriptor(helloworld.File_service_proto),
    sdk.WithFileDescriptor(echo.File_service_v1_proto),
)
require.NoError(t, err)

mock.Stub("helloworld.Greeter", "SayHello").
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()
mock.Stub("com.bavix.echo.v1.EchoService", "SendMessage").
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
| `Remote(grpcAddr)` | Connect to an external GripMock (gRPC + REST) |
| `WithSession(id)` | Session isolation for parallel tests (remote only) |

## Non-test / Remote mode

```go
// Run(nil, ...) — for benchmarks or when auto-cleanup is not applicable
mock, err := sdk.Run(nil, sdk.Remote("localhost:4770"), sdk.WithFileDescriptor(...))
if err != nil {
    log.Fatal(err)
}
defer func() { _ = mock.Close() }()

mock.Stub("helloworld.Greeter", "SayHello").
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()

client := helloworld.NewGreeterClient(mock.Conn())
```

## Installation

```bash
go get github.com/bavix/gripmock/v3/pkg/sdk
```
