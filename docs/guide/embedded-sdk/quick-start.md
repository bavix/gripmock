# Quick Start <VersionTag version="v3.7.0" />

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" />. Current API since <VersionTag version="v3.16.0" />; the legacy `sdk.Run` / `mock.Stub` / `mock.Verify` API was **removed in v3.20.0**. See the [Upgrade Guide](./upgrade.md).

Get started with GripMock Embedded SDK in your tests.

## Basic Example

Here's a simple example:

```go
import (
    "testing"

    "github.com/stretchr/testify/require"
    sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

func TestMyService_Call(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    defer srv.Close()

    // Define an expectation
    srv.ExpectUnary("/helloworld.Greeter/SayHello").
        Match("name", "Alex").
        Return("message", "Hi Alex")

    client := helloworld.NewGreeterClient(srv.Conn())

    // ACT
    reply, err := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Alex"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "Hi Alex", reply.GetMessage())
}
```

`NewServer` creates an independent server (safe for `t.Parallel()`), starts it on a random TCP port, and registers `t.Cleanup` for auto-verify + close. There is no error return — programmer mistakes panic immediately.

## Test Helper Pattern

For better organization, create a helper function:

```go
func runMyServiceMock(t *testing.T) *sdk.Server {
    t.Helper()

    return sdk.NewServer(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
}

func TestMyService_WithHelper(t *testing.T) {
    // ARRANGE
    srv := runMyServiceMock(t)

    srv.ExpectUnary("/helloworld.Greeter/SayHello").
        Match("name", "Alex").
        Return("message", "Hi Alex")

    client := helloworld.NewGreeterClient(srv.Conn())

    // ACT
    reply, err := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Alex"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "Hi Alex", reply.GetMessage())
}
```

## More Complex Example

Here's a more complex example with delay:

```go
import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

func TestGreeter_SayHello_WithDelay(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    defer srv.Close()

    // Define a stub with a delay, using the composable sdk.Delay()
    srv.ExpectUnary("/helloworld.Greeter/SayHello").
        Match("name", "Bob").
        Return(sdk.Delay(20*time.Millisecond, "message", "Hello Bob"))

    client := helloworld.NewGreeterClient(srv.Conn())

    // ACT
    start := time.Now()
    reply, err := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Bob"})
    elapsed := time.Since(start)

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "Hello Bob", reply.GetMessage())
    require.GreaterOrEqual(t, elapsed.Milliseconds(), int64(20))
}
```


## Using Full Method Constants

```go
srv.ExpectUnary(helloworld.Greeter_SayHello_FullMethodName).
    Match("name", "Alex").
    Return("message", "Hi Alex")
```
