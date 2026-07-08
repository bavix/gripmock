# Advanced Features <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** All features existed in <VersionTag version="v3.7.0" /> (legacy API). Current v2 API since <VersionTag version="v3.16.0" />. See the [Upgrade Guide](./upgrade.md) for migration.

Learn about advanced features of the GripMock Embedded SDK.

---

### Legacy API (v3.7.0+)

The same features in the legacy API:

```go
mock.Stub(sdk.By(FullMethod)).
    WhenHeaders(sdk.Equals("authorization", "Bearer token")).
    Reply(sdk.Data("result", "ok")).
    Delay(100 * time.Millisecond).
    Priority(10).
    Times(3).
    Commit()
```

---

## Headers Matching

```go
func TestAuthService_AuthenticatedEndpoint(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(auth.File_auth_service_proto))
    defer srv.Close()

    srv.ExpectUnary(AuthService_ProtectedEndpoint_FullMethodName).
        Match("resource", "secret-data").
        WithHeader(sdk.Contains("authorization", "Bearer valid-token")).
        Return("data", "secret-content")

    srv.ExpectUnary(AuthService_ProtectedEndpoint_FullMethodName).
        Match("resource", "secret-data").
        WithHeader(sdk.Contains("authorization", "Bearer invalid-token")).
        ReturnError(codes.Unauthenticated, "Invalid token")

    client := NewAuthServiceClient(srv.Conn())

    // ACT - Valid token
    validCtx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer valid-token"))
    validReply, validErr := client.ProtectedEndpoint(validCtx, &ProtectedEndpointRequest{Resource: "secret-data"})

    // ACT - Invalid token
    invalidCtx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer invalid-token"))
    _, invalidErr := client.ProtectedEndpoint(invalidCtx, &ProtectedEndpointRequest{Resource: "secret-data"})

    // ASSERT
    require.NoError(t, validErr)
    require.Equal(t, "secret-content", validReply.GetData())
    require.Error(t, invalidErr)
    require.Equal(t, codes.Unauthenticated, status.Code(invalidErr))
}
```

## Delays

```go
func TestExternalService_SlowResponse(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(external.File_external_service_proto))
    defer srv.Close()

    srv.ExpectUnary(ExternalService_Process_FullMethodName).
        Match("id", "slow-request").
        Return(Delay(500*time.Millisecond, "result", "processed"))

    client := NewExternalServiceClient(srv.Conn())

    // ACT
    start := time.Now()
    reply, err := client.Process(t.Context(), &ProcessRequest{Id: "slow-request"})
    elapsed := time.Since(start)

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "processed", reply.GetResult())
    require.GreaterOrEqual(t, elapsed, 500*time.Millisecond)
}
```

## Priority

```go
func TestUserService_GetUser_Priority(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(user.File_user_service_proto))
    defer srv.Close()

    // High priority: Specific case
    srv.ExpectUnary(UserService_GetUser_FullMethodName).
        Match("id", "special-user").
        Priority(100).
        Return("name", "Special User", "role", "admin")

    // Lower priority: General case
    srv.ExpectUnary(UserService_GetUser_FullMethodName).
        Match(sdk.Contains("id", "")). // Matches any ID
        Priority(10).
        Return("name", "General User", "role", "user")

    client := NewUserServiceClient(srv.Conn())

    // ACT
    specialReply, err1 := client.GetUser(t.Context(), &GetUserRequest{Id: "special-user"})
    generalReply, err2 := client.GetUser(t.Context(), &GetUserRequest{Id: "regular-user"})

    // ASSERT
    require.NoError(t, err1)
    require.NoError(t, err2)
    require.Equal(t, "Special User", specialReply.GetName())
    require.Equal(t, "admin", specialReply.GetRole())
    require.Equal(t, "General User", generalReply.GetName())
    require.Equal(t, "user", generalReply.GetRole())
}
```

## Call Limits (Times)

```go
func TestRateLimitService_LimitedCalls(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(rate.File_rate_limit_service_proto))
    defer srv.Close()

    // Stub matches exactly 3 times, then becomes unavailable
    srv.ExpectUnary(RateLimitService_Call_FullMethodName).
        Match("id", "limited").
        Times(3).
        Return("result", "ok")

    client := NewRateLimitServiceClient(srv.Conn())

    // ACT
    reply1, err1 := client.Call(t.Context(), &CallRequest{Id: "limited"})
    reply2, err2 := client.Call(t.Context(), &CallRequest{Id: "limited"})
    reply3, err3 := client.Call(t.Context(), &CallRequest{Id: "limited"})
    _, err4 := client.Call(t.Context(), &CallRequest{Id: "limited"}) // Should fail

    // ASSERT
    require.NoError(t, err1)
    require.NoError(t, err2)
    require.NoError(t, err3)
    require.Error(t, err4) // Should fail after 3 calls
    require.Equal(t, "ok", reply1.GetResult())
    require.Equal(t, "ok", reply2.GetResult())
    require.Equal(t, "ok", reply3.GetResult())
}
```

## Streaming Support

```go
func TestChatService_ServerStreaming(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(chat.File_chat_service_proto))
    defer srv.Close()

    srv.ExpectServerStream(ChatService_ChatStream_FullMethodName).
        Match("roomId", "room-123").
        SendStream(
            map[string]any{"message": "Hello", "sender": "Alice"},
            map[string]any{"message": "Hi there", "sender": "Bob"},
            map[string]any{"message": "Goodbye", "sender": "Alice"},
        )

    client := NewChatServiceClient(srv.Conn())

    // ACT
    stream, err := client.ChatStream(t.Context(), &ChatStreamRequest{RoomId: "room-123"})
    require.NoError(t, err)

    var messages []ChatMessage
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            break
        }
        require.NoError(t, err)
        messages = append(messages, *msg)
    }

    // ASSERT
    require.Len(t, messages, 3)
    require.Equal(t, "Hello", messages[0].GetMessage())
    require.Equal(t, "Alice", messages[0].GetSender())
    require.Equal(t, "Hi there", messages[1].GetMessage())
    require.Equal(t, "Bob", messages[1].GetSender())
    require.Equal(t, "Goodbye", messages[2].GetMessage())
    require.Equal(t, "Alice", messages[2].GetSender())
}
```

## Error Responses

```go
func TestPaymentService_Failure(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(payment.File_payment_service_proto))
    defer srv.Close()

    srv.ExpectUnary(PaymentService_Charge_FullMethodName).
        Match("amount", 0).
        ReturnError(codes.InvalidArgument, "Amount must be greater than 0")

    client := NewPaymentServiceClient(srv.Conn())

    // ACT
    _, err = client.Charge(t.Context(), &ChargeRequest{Amount: 0})

    // ASSERT
    require.Error(t, err)
    require.Equal(t, codes.InvalidArgument, status.Code(err))
    require.Contains(t, err.Error(), "Amount must be greater than 0")
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::
