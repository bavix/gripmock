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

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "test")).
        Reply(sdk.Data("result", "success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT - Make calls to the mock
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})

    // ASSERT - Verify the method was called exactly 2 times
    mock.Verify().Method("MyService", "MyMethod").Called(t, 2)
}
```

## Total Call Verification

```go
func TestMyService_TotalCalls(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)

    mock.Stub("MyService", "MethodA").
        When(sdk.Equals("id", "a")).
        Reply(sdk.Data("result", "A")).
        Commit()

    mock.Stub("MyService", "MethodB").
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

    mock.Stub("MyService", "MyMethod").
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
    
    // Check the request data
    require.Equal(t, "tracked", calls[0].Request["id"])
    require.Equal(t, "tracked", calls[1].Request["id"])
    
    // Verify individual calls
    mock.Verify().Method("MyService", "MyMethod").Called(t, 2)
}
```

## Never Called Verification

```go
func TestMyService_NeverCalled(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)

    mock.Stub("MyService", "UsedMethod").
        When(sdk.Equals("id", "used")).
        Reply(sdk.Data("result", "success")).
        Commit()

    // Don't define stub for UnusedMethod - it shouldn't be called

    client := NewMyServiceClient(mock.Conn())

    // ACT - Only call the used method
    _, _ = client.UsedMethod(t.Context(), &UsedMethodRequest{Id: "used"})

    // ASSERT - Verify the unused method was never called
    mock.Verify().Method("MyService", "UnusedMethod").Never(t)
    
    // Also verify the used method was called once
    mock.Verify().Method("MyService", "UsedMethod").Called(t, 1)
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
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "limited")).
        Reply(sdk.Data("result", "ok")).
        Times(2). // Expected to be called exactly 2 times
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT - Make 2 calls (the exact number specified in Times)
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "limited"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "limited"})
    
    // ASSERT - When t is passed to Run, verification happens automatically at test cleanup
    // If the actual call count doesn't match the Times value, the test will fail
    // The test will pass because we made exactly 2 calls as specified in Times(2)
}
```

## Complex Verification Scenario

```go
func TestOrderService_ComplexVerification(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(order.File_order_service_proto))
    require.NoError(t, err)

    // Create stubs with different call limits
    mock.Stub("OrderService", "CreateOrder").
        When(sdk.Equals("userId", "premium")).
        Reply(sdk.Data("orderId", "ORD-001")).
        Times(1). // Should be called exactly once
        Commit()

    mock.Stub("OrderService", "CancelOrder").
        When(sdk.Equals("orderId", "ORD-001")).
        Reply(sdk.Data("status", "cancelled")).
        Times(1). // Should be called exactly once
        Commit()

    mock.Stub("OrderService", "GetOrder").
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

    // ASSERT - Verification happens automatically due to Times() and passing t to Run()
    // All verifications will pass because we made the exact number of calls specified
    require.Equal(t, "ORD-001", createResp.GetOrderId())
}
```

## Verification with History Analysis

```go
func TestPaymentService_HistoryAnalysis(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(payment.File_payment_service_proto))
    require.NoError(t, err)

    mock.Stub("PaymentService", "ProcessPayment").
        When(sdk.Equals("amount", 100)).
        Reply(sdk.Data("transactionId", "TXN-100")).
        Commit()

    mock.Stub("PaymentService", "ProcessPayment").
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
    mock.Verify().Method("PaymentService", "ProcessPayment").Called(t, 3)
    
    // Analyze history
    allCalls := mock.History().All()
    require.Len(t, allCalls, 3)
    
    // Verify specific call details
    require.Equal(t, float64(100), allCalls[0].Request["amount"])
    require.Equal(t, "TXN-100", allCalls[0].Response["transactionId"])
    require.Equal(t, float64(200), allCalls[1].Request["amount"])
    require.Equal(t, "TXN-200", allCalls[1].Response["transactionId"])
    require.Equal(t, float64(100), allCalls[2].Request["amount"])
    require.Equal(t, "TXN-100", allCalls[2].Response["transactionId"])
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::