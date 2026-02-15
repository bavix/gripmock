# Best Practices <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

Recommended patterns and practices for using GripMock Embedded SDK.

## Test Helper Functions

Create reusable helper functions for common mock setup:

```go
func runMyServiceMock(t *testing.T, opts ...sdk.Option) (sdk.Mock, MyServiceClient) {
    t.Helper()
    
    // ARRANGE
    // Add default options
    allOpts := []sdk.Option{
        sdk.WithFileDescriptor(service.File_service_proto),
    }
    allOpts = append(allOpts, opts...)
    
    mock, err := sdk.Run(t, allOpts...)
    if err != nil {
        t.Fatalf("Failed to start GripMock: %v", err)
    }
    
    client := NewMyServiceClient(mock.Conn())
    return mock, client
}

func TestMyService_WithHelper(t *testing.T) {
    // ARRANGE
    mock, client := runMyServiceMock(t)
    
    // Define stubs in the Arrange phase
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "test")).
        Reply(sdk.Data("result", "success")).
        Commit()

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "test"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "success", resp.Result)
}
```

## Parallel Tests

Use sessions for parallel tests when using remote mode:

```go
func TestMyService_Parallel(t *testing.T) {
    t.Parallel()
    
    // ARRANGE
    mock, err := sdk.Run(t,
        sdk.Remote("localhost:4770"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()), // Isolate this test's stubs
    )
    require.NoError(t, err)
    
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "parallel")).
        Reply(sdk.Data("result", "parallel-success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "parallel"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "parallel-success", resp.Result)
}
```

## Proper Cleanup

When not passing `t` to `Run`, ensure manual cleanup:

```go
// When using Run(nil, ...) - for non-test usage
func TestNonTestUsage(t *testing.T) { // Using t for assertions only
    // ARRANGE
    mock, err := sdk.Run(nil, sdk.WithFileDescriptor(service.File_service_proto)) // Note: nil instead of t
    if err != nil {
        t.Fatalf("Failed to start GripMock: %v", err)
    }
    defer func() { _ = mock.Close() }() // Manual cleanup required

    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "manual")).
        Reply(sdk.Data("result", "manual-success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "manual"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "manual-success", resp.Result)
}
```

## Verify Expected Calls

Always verify that your code makes the expected calls:

```go
func TestMyService_WithVerification(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)
    
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "test")).
        Reply(sdk.Data("result", "success")).
        Commit()

    client := NewMyServiceClient(mock.Conn())

    // ACT
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})
    
    // ASSERT - Verify the expected call was made exactly 2 times
    mock.Verify().Method("MyService", "MyMethod").Called(t, 2)
}
```

## Use Descriptive Stub IDs

When working with complex stubs, consider organizing with clear structure:

```go
func TestUserService_ComplexScenario(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(user.File_user_service_proto))
    require.NoError(t, err)
    
    // Existing user stub
    mock.Stub("UserService", "GetUser").
        When(sdk.Equals("id", "existing-user")).
        Reply(sdk.Data("name", "John Doe", "email", "john@example.com")).
        Commit()
    
    // Missing user stub
    mock.Stub("UserService", "GetUser").
        When(sdk.Equals("id", "missing-user")).
        ReplyError(codes.NotFound, "User not found").
        Commit()

    client := NewUserServiceClient(mock.Conn())

    // ACT
    existingResp, _ := client.GetUser(t.Context(), &GetUserRequest{Id: "existing-user"})
    _, missingErr := client.GetUser(t.Context(), &GetUserRequest{Id: "missing-user"})

    // ASSERT
    require.Equal(t, "John Doe", existingResp.GetName())
    require.Equal(t, "john@example.com", existingResp.GetEmail())
    require.Error(t, missingErr)
    require.Equal(t, codes.NotFound, status.Code(missingErr))
}
```

## Error Handling

Always check for errors when starting the mock:

```go
func runSafeMock(t *testing.T) (sdk.Mock, MyServiceClient) {
    t.Helper()
    
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err, "Failed to start GripMock - check proto file path and syntax")
    
    client := NewMyServiceClient(mock.Conn())
    return mock, client
}

func TestMyService_WithSafeMock(t *testing.T) {
    // ARRANGE
    mock, client := runSafeMock(t)
    
    mock.Stub("MyService", "MyMethod").
        When(sdk.Equals("id", "safe-test")).
        Reply(sdk.Data("result", "safe-success")).
        Commit()

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "safe-test"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "safe-success", resp.Result)
}
```

## Use Times for Exact Call Verification

When you need to verify exact call counts, use the Times feature:

```go
func TestRetryLogic_WithTimes(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
    require.NoError(t, err)
    
    // First 2 calls fail, 3rd succeeds (simulating retry logic)
    mock.Stub("ExternalService", "Call").
        When(sdk.Equals("attempt", "fail")).
        ReplyError(codes.Unavailable, "Service unavailable").
        Times(2). // Allow this stub to match exactly 2 times
        Commit()
        
    mock.Stub("ExternalService", "Call").
        When(sdk.Equals("attempt", "success")).
        Reply(sdk.Data("result", "success")).
        Commit()

    client := NewExternalServiceClient(mock.Conn())

    // ACT
    // First two calls will fail (triggering retries)
    _, err1 := client.Call(t.Context(), &CallRequest{Attempt: "fail"})
    _, err2 := client.Call(t.Context(), &CallRequest{Attempt: "fail"})
    
    // Third call will succeed
    successResp, err3 := client.Call(t.Context(), &CallRequest{Attempt: "success"})

    // ASSERT
    require.Error(t, err1)
    require.Error(t, err2)
    require.NoError(t, err3)
    require.Equal(t, "success", successResp.GetResult())
    
    // Verification happens automatically due to Times(2) and passing t to Run
}
```

## Avoid Over-Mocking

Only mock what you need to test:

```go
func TestPaymentService_WithMinimalMocks(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(payment.File_payment_service_proto))
    require.NoError(t, err)
    
    // Good: Only mock the service you're testing
    mock.Stub("PaymentService", "Charge").
        When(sdk.Equals("amount", 100)).
        Reply(sdk.Data("transactionId", "tx-123")).
        Commit()

    client := NewPaymentServiceClient(mock.Conn())

    // ACT
    resp, err := client.Charge(t.Context(), &ChargeRequest{Amount: 100})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "tx-123", resp.GetTransactionId())
}
```

## Comprehensive Example

Here's a complete example showing all best practices:

```go
func TestOrderService_Comprehensive(t *testing.T) {
    // ARRANGE
    mock, err := sdk.Run(t, sdk.WithFileDescriptor(order.File_order_service_proto))
    require.NoError(t, err)
    
    // Setup multiple stubs with different behaviors
    mock.Stub("OrderService", "CreateOrder").
        When(sdk.Equals("userId", "premium")).
        Reply(sdk.Data("orderId", "ORD-001", "status", "created")).
        Commit()
    
    mock.Stub("OrderService", "GetOrder").
        When(sdk.Equals("orderId", "ORD-001")).
        Reply(sdk.Data("status", "processing", "total", 99.99)).
        Times(2). // Expected to be called exactly 2 times
        Commit()
    
    mock.Stub("OrderService", "CancelOrder").
        When(sdk.Equals("orderId", "ORD-001")).
        Reply(sdk.Data("status", "cancelled")).
        Commit()

    client := NewOrderServiceClient(mock.Conn())

    // ACT
    // Create an order
    createResp, err := client.CreateOrder(t.Context(), &CreateOrderRequest{UserId: "premium"})
    require.NoError(t, err)
    
    // Check order status twice
    status1, err := client.GetOrder(t.Context(), &GetOrderRequest{OrderId: "ORD-001"})
    require.NoError(t, err)
    status2, err := client.GetOrder(t.Context(), &GetOrderRequest{OrderId: "ORD-001"})
    require.NoError(t, err)
    
    // Cancel the order
    cancelResp, err := client.CancelOrder(t.Context(), &CancelOrderRequest{OrderId: "ORD-001"})
    require.NoError(t, err)

    // ASSERT
    require.Equal(t, "ORD-001", createResp.GetOrderId())
    require.Equal(t, "created", createResp.GetStatus())
    require.Equal(t, "processing", status1.GetStatus())
    require.Equal(t, 99.99, status1.GetTotal())
    require.Equal(t, "processing", status2.GetStatus())
    require.Equal(t, 99.99, status2.GetTotal())
    require.Equal(t, "cancelled", cancelResp.GetStatus())
    
    // Verify call counts
    mock.Verify().Method("OrderService", "CreateOrder").Called(t, 1)
    mock.Verify().Method("OrderService", "GetOrder").Called(t, 2) // Due to Times(2)
    mock.Verify().Method("OrderService", "CancelOrder").Called(t, 1)
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::