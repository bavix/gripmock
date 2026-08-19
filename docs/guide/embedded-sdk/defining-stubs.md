# Defining Stubs <VersionTag version="v3.7.0" />

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" />. Current API since <VersionTag version="v3.16.0" />; the legacy `sdk.Run` / `mock.Stub` / `mock.Verify` API was **removed in v3.20.0**. See the [Upgrade Guide](./upgrade.md).

The SDK provides helper functions to define stubs easily.


## Basic Matching

```go
func TestUserService_GetUser(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(user.File_user_service_proto))
    defer srv.Close()

    // Define stubs in the Arrange phase
    srv.ExpectUnary(UserService_GetUser_FullMethodName).
        Match("id", "user-123").
        Return("name", "John Doe", "email", "john@example.com")

    client := NewUserServiceClient(srv.Conn())

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
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(user.File_user_service_proto))
    defer srv.Close()

    // Exact match stub
    srv.ExpectUnary(UserService_SearchUsers_FullMethodName).
        Match("name", "exact-match").
        Return("results", []any{
            map[string]any{"id": "1", "name": "exact-match"},
        })

    // Partial match stub
    srv.ExpectUnary(UserService_SearchUsers_FullMethodName).
        Match(sdk.Contains("name", "partial")).
        Return("results", []any{
            map[string]any{"id": "2", "name": "partial-result"},
        })

    // Regex match stub
    srv.ExpectUnary(UserService_SearchUsers_FullMethodName).
        Match(sdk.Matches("email", `^[a-zA-Z0-9._%+-]+@example\.com$`)).
        Return("results", []any{
            map[string]any{"id": "3", "name": "regex-match"},
        })

    client := NewUserServiceClient(srv.Conn())

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
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    defer srv.Close()

    // Define a stub with delay
    srv.ExpectUnary(Greeter_SayHello_FullMethodName).
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

## Dynamic Template Example

```go
func TestGreeter_SayHello_DynamicTemplate(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(helloworld.File_helloworld_proto))
    defer srv.Close()

    // Use dynamic template to return request data in response
    srv.ExpectUnary(Greeter_SayHello_FullMethodName).
        Match(sdk.Matches("name", ".+")).
        Return("message", "Hi {{.Request.name}}")

    client := helloworld.NewGreeterClient(srv.Conn())

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
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(auth.File_auth_service_proto))
    defer srv.Close()

    // With headers matching
    srv.ExpectUnary(AuthService_Login_FullMethodName).
        Match("username", "test-user").
        WithHeader(sdk.Equals("authorization", "Bearer valid-token")).
        Return("token", "jwt-token-here")

    // With call limit (Times)
    srv.ExpectUnary(AuthService_Login_FullMethodName).
        Match("username", "limited-user").
        Times(2).
        Return("token", "limited-token")

    client := NewAuthServiceClient(srv.Conn())

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
