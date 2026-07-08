# Verification <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Verification available since <VersionTag version="v3.7.0" /> (legacy API: `mock.Verify().Method(fm).Called(t, n)`, `mock.Verify().Total(t, n)`). Current v2 API since <VersionTag version="v3.16.0" />. See the [Upgrade Guide](./upgrade.md) for migration.

Verify that your code interacts with the mock as expected.

---

### Legacy API (v3.7.0+)

The same verification in the legacy API:

```go
mock, err := sdk.Run(t, sdk.WithFileDescriptor(service.File_service_proto))
require.NoError(t, err)
defer mock.Close()

mock.Stub(sdk.By(MyService_MyMethod_FullMethodName)).
    When(sdk.Equals("id", "test")).
    Reply(sdk.Data("result", "success")).
    Commit()

// ... make calls ...

mock.Verify().Method(sdk.By(MyService_MyMethod_FullMethodName)).Called(t, 2)
mock.Verify().Total(t, 3)
mock.History().FilterByMethod("MyService", "MyMethod")
```

---

## Call Verification

```go
func TestMyService_CallVerification(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "test").
        Return("result", "success")

    client := NewMyServiceClient(srv.Conn())

    // ACT - Make calls to the mock
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "test"})

    // ASSERT - Verify the method was called exactly 2 times
    require.Equal(t, 2, srv.Called(MyService_MyMethod_FullMethodName))
}
```

## Using Generated Full Method Constants <VersionTag version="v3.16.0" />

If your generated gRPC package exposes `*_FullMethodName` constants, you can avoid manual full-method strings:

```go
srv.ExpectUnary(myservice.MyService_MyMethod_FullMethodName).
    Match("id", "test").
    Return("result", "ok")

require.Equal(t, 1, srv.Called(myservice.MyService_MyMethod_FullMethodName))
```

## Total Call Verification

```go
func TestMyService_TotalCalls(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    srv.ExpectUnary(MyService_MethodA_FullMethodName).
        Match("id", "a").
        Return("result", "A")

    srv.ExpectUnary(MyService_MethodB_FullMethodName).
        Match("id", "b").
        Return("result", "B")

    client := NewMyServiceClient(srv.Conn())

    // ACT - Make calls to different methods
    _, _ = client.MethodA(t.Context(), &MethodARequest{Id: "a"})
    _, _ = client.MethodB(t.Context(), &MethodBRequest{Id: "b"})
    _, _ = client.MethodA(t.Context(), &MethodARequest{Id: "a"})

    // ASSERT - Verify total calls to all methods
    require.Equal(t, 3, srv.TotalCalls()) // 2 calls to MethodA + 1 call to MethodB
}
```

## Call History

```go
func TestMyService_History(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "tracked").
        Return("result", "ok")

    client := NewMyServiceClient(srv.Conn())

    // ACT - Make some calls
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "tracked"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "tracked"})

    // ASSERT - Check history
    calls := srv.History()
    require.Len(t, calls, 2)
    require.Len(t, calls[0].Requests, 1)
    require.Equal(t, "tracked", calls[0].Requests[0]["id"])
    require.Equal(t, "tracked", calls[1].Requests[0]["id"])

    // Verify individual calls
    require.Equal(t, 2, srv.Called(MyService_MyMethod_FullMethodName))
}
```

## Context-Aware Verification and History (Remote)

In remote mode, verification and history operations call GripMock REST endpoints.

- `srv.Called(fullMethod)` — returns call count for a method
- `srv.TotalCalls()` — returns total call count
- `srv.ExpectationsWereMet()` — verifies all expectations
- `srv.History()` — returns all recorded calls

All these APIs use `t.Context()` for their network requests.

If you need to pass a specific context (for cancellation, tracing, request-scoped values), use SDK context helpers:

```go
func TestMyService_RemoteContextAwareChecks(t *testing.T) {
    srv := sdk.NewServer(t,
        sdk.WithRemote("localhost:4770", "http://localhost:4771"),
        sdk.WithSession(t.Name()),
    )

    // ... Arrange/Act ...

    ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
    defer cancel()

    err = sdk.VerifyStubTimesErrContext(ctx, srv.ExpectationsWereMet())
    // ... or use the Server's context-aware method directly:
    err = srv.ExpectationsWereMetContext(ctx)
    require.NoError(t, err)

    calls, err := sdk.HistoryFilterByMethodContext(ctx, srv.History(), "MyService", "MyMethod")
    require.NoError(t, err)
    require.NotEmpty(t, calls)
}
```

These helpers are backward-compatible: for embedded mode they fall back to non-context APIs.

## Never Called Verification

```go
func TestMyService_NeverCalled(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    srv.ExpectUnary(MyService_UsedMethod_FullMethodName).
        Match("id", "used").
        Return("result", "success")

    // Don't define stub for UnusedMethod - it shouldn't be called

    client := NewMyServiceClient(srv.Conn())

    // ACT - Only call the used method
    _, _ = client.UsedMethod(t.Context(), &UsedMethodRequest{Id: "used"})

    // ASSERT - Verify the unused method was never called
    require.Equal(t, 0, srv.Called(MyService_UnusedMethod_FullMethodName))

    // Also verify the used method was called once
    require.Equal(t, 1, srv.Called(MyService_UsedMethod_FullMethodName))
}
```

## Automatic Verification with Times

When using the `Times` feature, the SDK automatically verifies that the expected number of calls were made:

```go
func TestMyService_TimesVerification(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(service.File_service_proto))

    // Stub that should be called exactly 2 times
    srv.ExpectUnary(MyService_MyMethod_FullMethodName).
        Match("id", "limited").
        Times(2). // Expected to be called exactly 2 times
        Return("result", "ok")

    client := NewMyServiceClient(srv.Conn())

    // ACT - Make exactly 2 calls as specified in Times(2)
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "limited"})
    _, _ = client.MyMethod(t.Context(), &MyRequest{Id: "limited"})

    // ASSERT - Automatic verification via srv.Cleanup
    // srv.Cleanup registers ExpectationsWereMet() — no manual call needed
    // To verify explicitly:
    require.NoError(t, srv.ExpectationsWereMet())
}
```

## Complex Verification Scenario

```go
func TestOrderService_ComplexVerification(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(order.File_order_service_proto))

    // Create stubs with different call limits
    srv.ExpectUnary(OrderService_CreateOrder_FullMethodName).
        Match("userId", "premium").
        Times(1). // Should be called exactly once
        Return("orderId", "ORD-001")

    srv.ExpectUnary(OrderService_CancelOrder_FullMethodName).
        Match("orderId", "ORD-001").
        Times(1). // Should be called exactly once
        Return("status", "cancelled")

    srv.ExpectUnary(OrderService_GetOrder_FullMethodName).
        Match("orderId", "ORD-001").
        Times(2). // Should be called exactly twice
        Return("status", "active")

    client := NewOrderServiceClient(srv.Conn())

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

    // ASSERT - Automatic verification via Times() in srv.Cleanup
    require.Equal(t, "ORD-001", createResp.GetOrderId())

    // Or verify explicitly:
    require.NoError(t, srv.ExpectationsWereMet())
}
```

## Verification with History Analysis

```go
func TestPaymentService_HistoryAnalysis(t *testing.T) {
    // ARRANGE
    srv := sdk.NewServer(t, sdk.WithFileDescriptor(payment.File_payment_service_proto))

    srv.ExpectUnary(PaymentService_ProcessPayment_FullMethodName).
        Match("amount", float64(100)).
        Return("transactionId", "TXN-100")

    srv.ExpectUnary(PaymentService_ProcessPayment_FullMethodName).
        Match("amount", float64(200)).
        Return("transactionId", "TXN-200")

    client := NewPaymentServiceClient(srv.Conn())

    // ACT
    _, _ = client.ProcessPayment(t.Context(), &ProcessPaymentRequest{Amount: 100})
    _, _ = client.ProcessPayment(t.Context(), &ProcessPaymentRequest{Amount: 200})
    _, _ = client.ProcessPayment(t.Context(), &ProcessPaymentRequest{Amount: 100})

    // ASSERT
    // Check total calls
    require.Equal(t, 3, srv.TotalCalls())

    // Check specific method calls
    require.Equal(t, 3, srv.Called(PaymentService_ProcessPayment_FullMethodName))

    // Analyze history
    allCalls := srv.History()
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
