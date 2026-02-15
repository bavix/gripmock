# Advanced Features <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

Learn about advanced features of the GripMock Embedded SDK.

## Headers Matching

```go
func TestAuthService_AuthenticatedEndpoint(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(auth.File_auth_service_proto))
    require.NoError(t, err)

    mock.Stub("AuthService", "ProtectedEndpoint").
        When(sdk.Equals("resource", "secret-data")).
        WhenHeaders(sdk.HeaderEquals("authorization", "Bearer valid-token")).
        Reply(sdk.Data("data", "secret-content")).
        Commit()

    mock.Stub("AuthService", "ProtectedEndpoint").
        When(sdk.Equals("resource", "secret-data")).
        WhenHeaders(sdk.HeaderEquals("authorization", "Bearer invalid-token")).
        ReplyError(codes.Unauthenticated, "Invalid token").
        Commit()

    client := NewAuthServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(external.File_external_service_proto))
    require.NoError(t, err)

    mock.Stub("ExternalService", "Process").
        When(sdk.Equals("id", "slow-request")).
        Reply(sdk.Data("result", "processed")).
        Delay(500 * time.Millisecond).
        Commit()

    client := NewExternalServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(user.File_user_service_proto))
    require.NoError(t, err)

    // High priority: Specific case
    mock.Stub("UserService", "GetUser").
        When(sdk.Equals("id", "special-user")).
        Reply(sdk.Data("name", "Special User", "role", "admin")).
        Priority(100).
        Commit()

    // Lower priority: General case
    mock.Stub("UserService", "GetUser").
        When(sdk.Contains("id", "")). // Matches any ID
        Reply(sdk.Data("name", "General User", "role", "user")).
        Priority(10).
        Commit()

    client := NewUserServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(rate.File_rate_limit_service_proto))
    require.NoError(t, err)

    // Stub matches exactly 3 times, then becomes unavailable
    mock.Stub("RateLimitService", "Call").
        When(sdk.Equals("id", "limited")).
        Reply(sdk.Data("result", "ok")).
        Times(3).
        Commit()

    client := NewRateLimitServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(chat.File_chat_service_proto))
    require.NoError(t, err)

    mock.Stub("ChatService", "ChatStream").
        When(sdk.Equals("roomId", "room-123")).
        ReplyStream(
            sdk.Data("message", "Hello", "sender", "Alice"),
            sdk.Data("message", "Hi there", "sender", "Bob"),
            sdk.Data("message", "Goodbye", "sender", "Alice"),
        ).
        Commit()

    client := NewChatServiceClient(mock.Conn())

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
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(payment.File_payment_service_proto))
    require.NoError(t, err)

    mock.Stub("PaymentService", "Charge").
        When(sdk.Equals("amount", 0)).
        ReplyError(codes.InvalidArgument, "Amount must be greater than 0").
        Commit()

    client := NewPaymentServiceClient(mock.Conn())

    // ACT
    _, err = client.Charge(t.Context(), &ChargeRequest{Amount: 0})

    // ASSERT
    require.Error(t, err)
    require.Equal(t, codes.InvalidArgument, status.Code(err))
    require.Contains(t, err.Error(), "Amount must be greater than 0")
}
```

## Response Headers

```go
func TestAuthService_LoginWithHeaders(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(auth.File_auth_service_proto))
    require.NoError(t, err)

    mock.Stub("AuthService", "Login").
        When(sdk.Equals("username", "test-user")).
        Reply(sdk.Data("token", "jwt-token")).
        ReplyHeaderPairs("x-session-id", "session-123", "x-permissions", "read,write").
        Commit()

    client := NewAuthServiceClient(mock.Conn())

    // ACT
    ctx := t.Context()
    reply, err := client.Login(ctx, &LoginRequest{Username: "test-user"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "jwt-token", reply.GetToken())
    
    // Check response headers
    trailer := metadata.MD{}
    require.NoError(t, grpc.GetTrailer(ctx, &trailer))
    require.Equal(t, []string{"session-123"}, trailer["x-session-id"])
    require.Equal(t, []string{"read,write"}, trailer["x-permissions"])
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::