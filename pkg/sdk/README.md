# GripMock SDK

Go SDK for embedding a gRPC mock server in tests or connecting to a remote GripMock instance.

> **⚠️ Experimental SDK**  
> This SDK is experimental and may be discontinued or never released. Use at your own risk.

## Quick start

```go
mock, err := sdk.Run(ctx, sdk.WithFileDescriptor(helloworld.File_service_proto))
if err != nil {
    log.Fatal(err)
}
defer mock.Close()

mock.Stub("helloworld.Greeter", "SayHello").
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()

client := helloworld.NewGreeterClient(mock.Conn())
reply, _ := client.SayHello(ctx, &helloworld.HelloRequest{Name: "Alex"})
// reply.Message == "Hi Alex"
```

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

## Run options

| Option | Description |
|--------|-------------|
| `WithFileDescriptor(fd)` | Use generated descriptor (e.g. `helloworld.File_service_proto`) |
| `WithDescriptors(fds)` | Use `FileDescriptorSet` from protoset |
| `WithListenAddr(network, addr)` | Listen on a real port (e.g. `"tcp", ":0"`) |
| `Remote(grpcAddr)` | Connect to an external GripMock (gRPC + REST) |
| `WithSession(id)` | Session isolation for parallel tests (remote only) |

## Remote mode

```go
mock, err := sdk.Run(ctx, sdk.Remote("localhost:4770"))
defer mock.Close()

mock.Stub("helloworld.Greeter", "SayHello").
    Unary("name", "Alex", "message", "Hi Alex").
    Commit()

client := helloworld.NewGreeterClient(mock.Conn())
```

## Installation

```bash
go get github.com/bavix/gripmock/v3/pkg/sdk
```
