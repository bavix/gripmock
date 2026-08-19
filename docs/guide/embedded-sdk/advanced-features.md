# Advanced Features <VersionTag version="v3.7.0" />

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" />. Current API since <VersionTag version="v3.16.0" />; the legacy `sdk.Run` / `mock.Stub` / `mock.Verify` API was **removed in v3.20.0**. See the [Upgrade Guide](./upgrade.md).

Learn about advanced features of the GripMock Embedded SDK.


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
        Return(sdk.Delay(500*time.Millisecond, "result", "processed"))

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
    _, err := client.Charge(t.Context(), &ChargeRequest{Amount: 0})

    // ASSERT
    require.Error(t, err)
    require.Equal(t, codes.InvalidArgument, status.Code(err))
    require.Contains(t, err.Error(), "Amount must be greater than 0")
}
```

## Response Metadata <VersionTag version="v3.20.0" />

Every expectation type can set the metadata the call answers with. Values go
through the template engine, so they can reference the request.

```go
srv.ExpectUnary(FullMethod).
    Match("id", "42").
    ReturnHeaders(map[string]string{"x-trace-id": "{{.Request.id}}"}).
    ReturnTrailers(map[string]string{"x-cost": "7"}).
    Return("name", "Alex")
```

## Fixed Stub IDs <VersionTag version="v3.20.0" />

`WithID` pins the identifier instead of generating one, so an effect or a later
test step can target the same stub deterministically.

```go
id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

srv.ExpectUnary(FullMethod).WithID(id).Match("id", "42").Return("name", "Alex")
```

## Scalar and Status-Only Responses <VersionTag version="v3.20.0" />

`Return` builds a message from key-value pairs. A method whose response is a
top-level well-known type needs `ReturnValue`, and a bare status needs
`ReturnStatus`.

```go
srv.ExpectUnary("/wkt.TypeService/GetName").Match("id", "1").ReturnValue("plain string")
srv.ExpectUnary(FullMethod).Match("id", "gone").ReturnStatus(codes.Unavailable)
```

A client-stream call answers with one message too, so it takes the same terminal
methods. `ReturnStatus`, `ReturnError` and `Delay` work on all four shapes;
`Delay` pauses the reply, or every message of a stream.

```go
srv.ExpectClientStream("/calc.Calculator/SumNumbers").Match(sdk.Matches("value", ".*")).ReturnStatus(codes.ResourceExhausted)
srv.ExpectBidirectionalStream("/chat.ChatService/Chat").Match("text", "ping").ReturnError(codes.FailedPrecondition, "room closed")
srv.ExpectUnary(FullMethod).Match("id", "slow").Delay(80 * time.Millisecond).Return("name", "late")
```

## Positional Stream Matching <VersionTag version="v3.20.0" />

`Match` merges its matchers and applies the result to every message.
`MatchSequence` matches each message against its own matcher.

```go
srv.ExpectClientStream("/calc.Calculator/SumNumbers").
    MatchSequence(sdk.Equals("value", 1.0), sdk.Equals("value", 2.0)).
    Return("result", 3.0, "count", 2)
```

## Aborting a Stream <VersionTag version="v3.20.0" />

A stream can end in an error either from the start or partway through.

```go
// Fails immediately.
srv.ExpectServerStream(FullMethod).Match("q", "boom").
    ReturnError(codes.ResourceExhausted, "quota exceeded")

// Delivers one message, then fails.
srv.ExpectServerStream(FullMethod).Match("q", "half").
    SendStream(
        map[string]any{"id": "1"},
        sdk.StreamError(codes.Aborted, "cut short"),
    )
```

`Delay(d)` on a server-stream expectation pauses before **every** message;
`SendStream(sdk.Delay(d, ...))` delays a single one.

## Static Bidirectional Stubs <VersionTag version="v3.20.0" />

`Run` installs an in-process handler and therefore only works against an
embedded server. A static bidi stub works in both modes.

```go
srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
    Match("text", "ping").
    SendStream(map[string]any{"text": "pong"})
```

## Handlers <VersionTag version="v3.20.0" />

Unary, server-stream and client-stream expectations accept a handler when a
static response is not enough. Handlers run in-process and **panic in remote
mode**.

```go
srv.ExpectUnary(FullMethod).
    Match("name", "Alex").
    Run(func(ctx context.Context, in any) (any, error) {
        req := in.(map[string]any)

        return map[string]any{"message": "hello " + req["name"].(string)}, nil
    })
```

A client-stream handler receives the request messages already decoded — the
engine has to drain the stream before it can pick a stub, so the handler cannot
read it again.

```go
srv.ExpectClientStream("/calc.Calculator/SumNumbers").
    Match(sdk.Matches("value", ".*")).
    Run(func(ctx context.Context, messages []any) (any, error) {
        return map[string]any{"count": len(messages)}, nil
    })
```
