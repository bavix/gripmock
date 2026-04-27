# Verification <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

Verify that your code interacts with the mock as expected.

## Call Verification

```go
func TestMyService_CallVerification(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)

    mock.Stub(sdk.By(MyService_MyMethod_FullMethodName)).
        When(sdk.Equals("id", "test")).
        Reply(sdk.Data("result", "success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT - Make calls to the mock
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})

    // ASSERT - Verify the method was called exactly 2 times
    mock.Verify().Method(sdk.By(MyService_MyMethod_FullMethodName)).Called(t, 2)
}
```

## Using Generated Full Method Constants <VersionTag version="v3.9.1" />

If your generated gRPC package exposes `*_FullMethodName` constants, you can avoid manual full-method strings:

```go
mock.Stub(sdk.By(myservice.MyService_MyMethod_FullMethodName)).
    When(sdk.Equals("id", "test")).
    Reply(sdk.Data("result", "ok")).
    Commit()

mock.Verify().Method(sdk.By(myservice.MyService_MyMethod_FullMethodName)).Called(t, 1)
```

## Total Call Verification

```go
func TestMyService_TotalCalls(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)

    mock.Stub(sdk.By(MyService_MethodA_FullMethodName)).
        When(sdk.Equals("id", "a")).
        Reply(sdk.Data("result", "A")).
        Commit()

    mock.Stub(sdk.By(MyService_MethodB_FullMethodName)).
        When(sdk.Equals("id", "b")).
        Reply(sdk.Data("result", "B")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT - Make calls to different methods
    _, _ = client.MethodA(t.Context(), &MethodARequest{Id: "a"})
    _, _ = client.MethodB(t.Context(), &MethodBRequest{Id: "b"})
    _, _ = client.MethodA(t.Context(), &MethodARequest{Id: "a"})

    // ASSERT - Verify total calls to all methods
    mock.Verify().Total(t, 3) // 2 calls to MethodA + 1 call to MethodB
}
```

## Call History

```go
func TestMyService_History(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)

    mock.Stub(sdk.By(MyService_MyMethod_FullMethodName)).
        When(sdk.Equals("id", "tracked")).
        Reply(sdk.Data("result", "ok")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT - Make some calls
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "tracked"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "tracked"})

    // ASSERT - Check history
    calls := mock.History().FilterByMethod("MyService", "MyMethod")
    require.Len(t, calls, 2)
    require.Len(t, calls[0].Requests, 1)
    require.Len(t, calls[1].Requests, 1)

    // Check the request data
    require.Equal(t, "tracked", calls[0].Requests[0]["id"])
    require.Equal(t, "tracked", calls[1].Requests[0]["id"])

    // Verify individual calls
    mock.Verify().Method(sdk.By(MyService_MyMethod_FullMethodName)).Called(t, 2)
}
```

## Context-Aware Verification and History (Remote)

In remote mode, verification and history operations call GripMock REST endpoints.

- `mock.Verify().Method(sdk.By(...)).Called(t, n)`
- `mock.Verify().Total(t, n)`
- `mock.Verify().VerifyStubTimes(t)`

All these APIs use `t.Context()` for their network requests.

If you need to pass a specific context (for cancellation, tracing, request-scoped values), use SDK context helpers:

```go
func TestMyService_RemoteContextAwareChecks(t *testing.T) {
    mock, err := sdk.Run(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession(t.Name()),
    )
    require.NoError(t, err)

    // ... Arrange/Act ...

    ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
    defer cancel()

    err = sdk.VerifyStubTimesErrContext(ctx, mock.Verify())
    require.NoError(t, err)

    calls, err := sdk.HistoryFilterByMethodContext(ctx, mock.History(), "MyService", "MyMethod")
    require.NoError(t, err)
    require.NotEmpty(t, calls)
}
```

These helpers are backward-compatible: for embedded mode they fall back to non-context APIs.

## Never Called Verification

```go
func TestMyService_NeverCalled(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)

    mock.Stub(sdk.By(MyService_UsedMethod_FullMethodName)).
        When(sdk.Equals("id", "used")).
        Reply(sdk.Data("result", "success")).
        Commit()

    // Don't define stub for UnusedMethod - it shouldn't be called

    client := NewMyServiceClient(mock.Conn())

    // ACT - Only call the used method
    _, _ = client.UsedMethod(t.Context(), &UsedMethodRequest{Id: "used"})

    // ASSERT - Verify the unused method was never called
    mock.Verify().Method(sdk.By(MyService_UnusedMethod_FullMethodName)).Never(t)

    // Also verify the used method was called once
    mock.Verify().Method(sdk.By(MyService_UsedMethod_FullMethodName)).Called(t, 1)
}
```

## Automatic Verification with Times

When using the `Times` feature, the SDK automatically verifies that the expected number of calls were made:

```go
func TestMyService_TimesVerification(t *testing.T) {
    // ARRANGE
    // Pass t to Run to enable automatic verification
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)

    // Stub that should be called exactly 2 times
    mock.Stub(sdk.By(MyService_MyMethod_FullMethodName)).
        When(sdk.Equals("id", "limited")).
        Reply(sdk.Data("result", "ok")).
        Times(2). // Expected to be called exactly 2 times
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT - Make exactly 2 calls as specified in Times(2)
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "limited"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "limited"})

    // ASSERT - Automatic verification at test cleanup via Times(2)
}
```

## Complex Verification Scenario

```go
func TestOrderService_ComplexVerification(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(order.File_order_service_proto))
    require.NoError(t, err)

    // Create stubs with different call limits
    mock.Stub(sdk.By(OrderService_CreateOrder_FullMethodName)).
        When(sdk.Equals("userId", "premium")).
        Reply(sdk.Data("orderId", "ORD-001")).
        Times(1). // Should be called exactly once
        Commit()

    mock.Stub(sdk.By(OrderService_CancelOrder_FullMethodName)).
        When(sdk.Equals("orderId", "ORD-001")).
        Reply(sdk.Data("status", "cancelled")).
        Times(1). // Should be called exactly once
        Commit()

    mock.Stub(sdk.By(OrderService_GetOrder_FullMethodName)).
        When(sdk.Equals("orderId", "ORD-001")).
        Reply(sdk.Data("status", "active")).
        Times(2). // Should be called exactly twice
        Commit()

    client := NewOrderServiceClient(mock.Conn())

    // ACT
    // Create order
    createResp, err := client.CreateOrder(t.Context(), &CreateOrderRequest{UserId: "premium"})
    require.NoError(t, err)

    // Check order status twice
    _, err = client.GetOrder(t.Context(), &GetOrderRequest{OrderId: "ORD-001"})
    require.NoError(t, err)
    _, err = client.GetOrder(t.Context(), &GetOrderRequest{OrderId: "ORD-001"})
    require.NoError(t, err)

    // Cancel order
    _, err = client.CancelOrder(t.Context(), &CancelOrderRequest{OrderId: "ORD-001"})
    require.NoError(t, err)

    // ASSERT - Automatic verification via Times()
    require.Equal(t, "ORD-001", createResp.GetOrderId())
}
```

## Verification with History Analysis

```go
func TestPaymentService_HistoryAnalysis(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(payment.File_payment_service_proto))
    require.NoError(t, err)

    mock.Stub(sdk.By(PaymentService_ProcessPayment_FullMethodName)).
        When(sdk.Equals("amount", 100)).
        Reply(sdk.Data("transactionId", "TXN-100")).
        Commit()

    mock.Stub(sdk.By(PaymentService_ProcessPayment_FullMethodName)).
        When(sdk.Equals("amount", 200)).
        Reply(sdk.Data("transactionId", "TXN-200")).
        Commit()

    client := NewPaymentServiceClient(mock.Conn())

    // ACT
    _, _ = client.ProcessPayment(t.Context(), &ProcessPaymentRequest{Amount: 100})
    _, _ = client.ProcessPayment(t.Context(), &ProcessPaymentRequest{Amount: 200})
    _, _ = client.ProcessPayment(t.Context(), &ProcessPaymentRequest{Amount: 100})

    // ASSERT
    // Check total calls
    mock.Verify().Total(t, 3)

    // Check specific method calls
    mock.Verify().Method(sdk.By(PaymentService_ProcessPayment_FullMethodName)).Called(t, 3)

    // Analyze history
    allCalls := mock.History().All()
    require.Len(t, allCalls, 3)

    for _, call := range allCalls {
        require.Len(t, call.Requests, 1)
        require.Len(t, call.Responses, 1)
    }

    // Verify specific call details
    require.Equal(t, float64(100), allCalls[0].Requests[0]["amount"])
    require.Equal(t, "TXN-100", allCalls[0].Responses[0]["transactionId"])
    require.Equal(t, float64(200), allCalls[1].Requests[0]["amount"])
    require.Equal(t, "TXN-200", allCalls[1].Responses[0]["transactionId"])
    require.Equal(t, float64(100), allCalls[2].Requests[0]["amount"])
    require.Equal(t, "TXN-100", allCalls[2].Responses[0]["transactionId"])
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::
