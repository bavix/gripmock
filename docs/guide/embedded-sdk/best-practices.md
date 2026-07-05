# Best Practices <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" /> (legacy API). Current v2 API since <VersionTag version="v3.16.0" />.

Recommended patterns and practices for using GripMock Embedded SDK.

## Test Helper Functions

Create reusable helper functions for common mock setup:

```go
func runMyServiceMock(t *testing.T, opts ...sdk.Option) *sdk.Server {
    t.Helper()

    // ARRANGE
    // Add default options
    allOpts := []sdk.Option{
        sdk.WithFileDescriptor(service.File_service_proto),
    }
    allOpts = append(allOpts, opts...)

    return sdk.NewServer(t, allOpts...)
}

func TestMyService_WithHelper(t *testing.T) {
    // ARRANGE
    srv := runMyServiceMock(t)

    // Define stubs in the Arrange phase
    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "test").
        Return("result", "success")

    client := NewMyServiceClient(srv.Conn())

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
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithFileDescriptor(service.File_service_proto),
        sdk.WithSession(t.Name()), // Isolate this test's stubs
    )

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "parallel").
        Return("result", "parallel-success")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "parallel"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "parallel-success", resp.Result)
}
```

## Proper Cleanup

Always pass `t` to `NewServer`. The SDK registers cleanup automatically and verifies `Times(...)` expectations:

```go
func TestCleanupIsAutomatic(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "manual").
        Return("result", "manual-success")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    resp, err := client.MyMethod(t.Context(), &MyRequest{Id: "manual"})

    // ASSERT
    require.NoError(t, err)
    require.Equal(t, "manual-success", resp.Result)

    // No explicit srv.Close() is required in tests.
}
```

## Verify Expected Calls

Always verify that your code makes the expected calls:

```go
func TestMyService_WithVerification(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "test").
        Return("result", "success")

    client := NewMyServiceClient(srv.Conn())

    // ACT
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})

    // ASSERT - Verify the expected call was made exactly 2 times
    require.Equal(t, 2, srv.Called(MyService_MyMethod_FullMethodName))
}
```

## Use Descriptive Stub IDs

When working with complex stubs, consider organizing with clear structure:

```go
func TestUserService_ComplexScenario(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(user.File_user_service_proto))

    // Existing user stub
    srv.ExpectUnary(UserService_GetUser_FullMethodName).
        Match("id", "existing-user").
        Return("name", "John Doe", "email", "john@example.com")

    // Missing user stub
    srv.ExpectUnary(UserService_GetUser_FullMethodName).
        Match("id", "missing-user").
        ReturnError(codes.NotFound, "User not found")

    client := NewUserServiceClient(srv.Conn())

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

The SDK's `NewServer` does not return an error — startup failures trigger a panic internally:

```go
func runSafeMock(t *testing.T) *sdk.Server {
    t.Helper()

    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    return srv
}

func TestMyService_WithSafeMock(t *testing.T) {
    // ARRANGE
    srv := runSafeMock(t)

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "safe-test").
        Return("result", "safe-success")

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
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    // First 2 calls fail, 3rd succeeds (simulating retry logic)
    srv.ExpectUnary(ExternalService_Call_FullMethodName).
        Match("attempt", "fail").
        Times(2). // Allow this stub to match exactly 2 times
        ReturnError(codes.Unavailable, "Service unavailable")

    srv.ExpectUnary(ExternalService_Call_FullMethodName).
        Match("attempt", "success").
        Return("result", "success")

    client := NewExternalServiceClient(srv.Conn())

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

    // Verification happens automatically due to Times(2) and passing t to NewServer
}
```

## Avoid Over-Mocking

Only mock what you need to test:

```go
func TestPaymentService_WithMinimalMocks(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(payment.File_payment_service_proto))

    // Good: Only mock the service you're testing
    srv.ExpectUnary(PaymentService_Charge_FullMethodName).
        Match("amount", 100).
        Return("transactionId", "tx-123")

    client := NewPaymentServiceClient(srv.Conn())

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
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(order.File_order_service_proto))

    // Setup multiple stubs with different behaviors
    srv.ExpectUnary(OrderService_CreateOrder_FullMethodName).
        Match("userId", "premium").
        Return("orderId", "ORD-001", "status", "created")

    srv.ExpectUnary(OrderService_GetOrder_FullMethodName).
        Match("orderId", "ORD-001").
        Times(2). // Expected to be called exactly 2 times
        Return("status", "processing", "total", 99.99)

    srv.ExpectUnary(OrderService_CancelOrder_FullMethodName).
        Match("orderId", "ORD-001").
        Return("status", "cancelled")

    client := NewOrderServiceClient(srv.Conn())

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
    require.Equal(t, 1, srv.Called(OrderService_CreateOrder_FullMethodName))
    require.Equal(t, 2, srv.Called(OrderService_GetOrder_FullMethodName)) // Due to Times(2)
    require.Equal(t, 1, srv.Called(OrderService_CancelOrder_FullMethodName))
}
```

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::
