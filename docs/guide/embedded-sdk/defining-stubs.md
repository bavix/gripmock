# Defining Stubs <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

The SDK provides helper functions to define stubs easily.

## Basic Matching

```go
func TestUserService_GetUser(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(user.File_user_service_proto))
    require.NoError(t, err)

    // Define stubs in the Arrange phase
    mock.Stub("UserService", "GetUser").
        When(sdk.Equals("id", "user-123")).
        Reply(sdk.Data("name", "John Doe", "email", "john@example.com")).
        Commit()

    client := NewUserServiceClient(mock.Conn())

    // ACT
    reply, err := client.GetUser(t.Context(), &GetUserRequest{Id: "user-123"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "John Doe", reply.GetName())
    require.Equal(t, "john@example.com", reply.GetEmail())
}
```

## Multiple Matching Strategies

```go
func TestUserService_SearchUsers(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(user.File_user_service_proto))
    require.NoError(t, err)

    // Exact match stub
    mock.Stub("UserService", "SearchUsers").
        When(sdk.Equals("name", "exact-match")).
        Reply(sdk.Data("results", []any{
            map[string]any{"id": "1", "name": "exact-match"},
        })).
        Commit()

    // Partial match stub
    mock.Stub("UserService", "SearchUsers").
        When(sdk.Contains("name", "partial")).
        Reply(sdk.Data("results", []any{
            map[string]any{"id": "2", "name": "partial-result"},
        })).
        Commit()

    // Regex match stub
    mock.Stub("UserService", "SearchUsers").
        When(sdk.Matches("email", `^[a-zA-Z0-9._%+-]+@example\.com$`)).
        Reply(sdk.Data("results", []any{
            map[string]any{"id": "3", "name": "regex-match"},
        })).
        Commit()

    client := NewUserServiceClient(mock.Conn())

    // ACT
    exactReply, err := client.SearchUsers(t.Context(), &SearchUsersRequest{Name: "exact-match"})
    require.NoError(t, err)

    partialReply, err := client.SearchUsers(t.Context(), &SearchUsersRequest{Name: "partial-search"})
    require.NoError(t, err)

    regexReply, err := client.SearchUsers(t.Context(), &SearchUsersRequest{Email: "test@example.com"})
    require.NoError(t, err)

    // ASSERT
    require.Equal(t, "1", exactReply.Results[0].Id)
    require.Equal(t, "2", partialReply.Results[0].Id)
    require.Equal(t, "3", regexReply.Results[0].Id)
}
```

## Real-World Example

Here's a more complete example based on actual usage:

```go
import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

func TestGreeter_SayHello(t *testing.T) {
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

## Dynamic Template Example

```go
func TestGreeter_SayHello_DynamicTemplate(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    require.NoError(t, err)
    
    // Use dynamic template to return request data in response
    mock.Stub("helloworld.Greeter", "SayHello").
        When(sdk.Matches("name", ".+")).
        Reply(sdk.Data("message", "Hi {{.Request.name}}")).
        Commit()

    client := helloworld.NewGreeterClient(mock.Conn())

    // ACT
    reply, err := client.SayHello(t.Context(), &helloworld.HelloRequest{Name: "Alex"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "Hi Alex", reply.GetMessage())
}
```

## Advanced Options

```go
func TestAuthService_Login(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(auth.File_auth_service_proto))
    require.NoError(t, err)

    // With headers matching
    mock.Stub("AuthService", "Login").
        When(sdk.Equals("username", "test-user")).
        WhenHeaders(sdk.HeaderEquals("authorization", "Bearer valid-token")).
        Reply(sdk.Data("token", "jwt-token-here")).
        Commit()

    // With call limit (Times)
    mock.Stub("AuthService", "Login").
        When(sdk.Equals("username", "limited-user")).
        Reply(sdk.Data("token", "limited-token")).
        Times(2). // Stub will only match 2 times
        Commit()

    client := NewAuthServiceClient(mock.Conn())

    // ACT
    authorizedReply, err := client.Login(t.Context(), &LoginRequest{Username: "test-user"})
    require.NoError(t, err)

    limitedReply1, err := client.Login(t.Context(), &LoginRequest{Username: "limited-user"})
    require.NoError(t, err)

    limitedReply2, err := client.Login(t.Context(), &LoginRequest{Username: "limited-user"})
    require.NoError(t, err)

    // Third call to limited-user should fail since Times(2)
    _, err = client.Login(t.Context(), &LoginRequest{Username: "limited-user"})
    require.Error(t, err)

    // ASSERT
    require.Equal(t, "jwt-token-here", authorizedReply.GetToken())
    require.Equal(t, "limited-token", limitedReply1.GetToken())
    require.Equal(t, "limited-token", limitedReply2.GetToken())
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::