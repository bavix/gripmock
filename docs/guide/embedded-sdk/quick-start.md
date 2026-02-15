# Quick Start <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

Get started with GripMock Embedded SDK in your tests.

## Basic Example

Here's a simple example of how to use the Embedded SDK:

```go
import (
    "testing"

    "github.com/stretchr/testify/require"
    sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

func TestMyService_Call(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    require.NoError(t, err)

    // Define a stub
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "test-id")).
        Reply(sdk.Data("result", "success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    reply, err := client.MyMethod(t.Context(), &MyRequest{Id: "test-id"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "success", reply.GetResult())
}
```

## Test Helper Pattern

For better organization, create a helper function:

```go
func runMyServiceMock(t *testing.T) (sdk.Mock, MyServiceClient) {
    t.Helper()
    
    // ARRANGE: Start embedded GripMock instance - pass t directly
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    require.NoError(t, err)

    client := NewMyServiceClient(mock.Conn())
    return mock, client
}

func TestMyService_WithHelper(t *testing.T) {
    // ARRANGE
    mock, client := runMyServiceMock(t)
    
    // Define a stub
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "test-id")).
        Reply(sdk.Data("result", "success")).
        Commit()

    // ACT
    reply, err := client.MyMethod(t.Context(), &MyRequest{Id: "test-id"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "success", reply.GetResult())
}
```

## More Complex Example

Here's a more complex example:

```go
import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

func TestGreeter_SayHello_WithDelay(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    require.NoError(t, err)

    // Define a stub with delay
    delayMs := 20
    mock.Stub("helloworld.Greeter", "SayHello").
        When(sdk.Equals("name", "Bob")).
        Reply(sdk.Data("message", "Hello Bob")).
        Delay(time.Duration(delayMs) * time.Millisecond).
        Commit()

    client := helloworld.NewGreeterClient(mock.Conn())

    // ACT
    start := time.Now()
    reply, err := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Bob"})
    elapsed := time.Since(start)

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "Hello Bob", reply.GetMessage())
    require.GreaterOrEqual(t, elapsed.Milliseconds(), int64(delayMs))
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::