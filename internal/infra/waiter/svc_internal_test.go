package waiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testServiceName = "test-service"

// testContextKey is a custom type for context keys to avoid collisions.
type testContextKey string

const testKey testContextKey = "key"

func TestNewService(t *testing.T) {
	// Test service creation with nil client
	service := NewService(nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.client)
}

func TestService_Struct(t *testing.T) {
	// Test Service struct initialization
	service := &Service{
		client: nil,
	}
	assert.NotNil(t, service)
	assert.Nil(t, service.client)
}

func TestService_PingWithTimeout_NilClient(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test ping with nil client (should panic or return error)
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_NilClient(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	serviceName := testServiceName

	// Test ping with nil client (should panic or return error)
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_ContextTimeout(t *testing.T) {
	service := NewService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)

	defer cancel()

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with already expired context
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_PingWithTimeout_ZeroTimeout(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	timeout := time.Duration(0)
	serviceName := testServiceName

	// Test with zero timeout
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_PingWithTimeout_NegativeTimeout(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	timeout := -1 * time.Second
	serviceName := testServiceName

	// Test with negative timeout
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_EmptyServiceName(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	serviceName := ""

	// Test with empty service name
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_Ping_NilContext(t *testing.T) {
	service := NewService(nil)

	var ctx context.Context = nil

	serviceName := testServiceName

	// Test with nil context
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_NilContext(t *testing.T) {
	service := NewService(nil)

	var ctx context.Context = nil

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with nil context
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_PingWithTimeout_CancelledContext(t *testing.T) {
	service := NewService(nil)
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with cancelled context
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_CancelledContext(t *testing.T) {
	service := NewService(nil)
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately

	serviceName := testServiceName

	// Test with cancelled context
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_VeryLongTimeout(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	timeout := 24 * time.Hour // Very long timeout
	serviceName := testServiceName

	// Test with very long timeout
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_PingWithTimeout_ContextWithDeadline(t *testing.T) {
	service := NewService(nil)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))

	defer cancel()

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with context that has already expired deadline
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_ContextWithValues(t *testing.T) {
	service := NewService(nil)
	ctx := context.WithValue(context.Background(), testKey, "value")
	serviceName := testServiceName

	// Test with context that has values
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_ContextWithValues(t *testing.T) {
	service := NewService(nil)
	ctx := context.WithValue(context.Background(), testKey, "value")
	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with context that has values
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_ServiceNameWithSpecialCharacters(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	serviceName := "test-service-with-special-chars-!@#$%^&*()"

	// Test with service name containing special characters
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_ServiceNameWithSpecialCharacters(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	timeout := 100 * time.Millisecond
	serviceName := "test-service-with-special-chars-!@#$%^&*()"

	// Test with service name containing special characters
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_UnicodeServiceName(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	serviceName := "тест-сервис-с-unicode"

	// Test with unicode service name
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_UnicodeServiceName(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	timeout := 100 * time.Millisecond
	serviceName := "тест-сервис-с-unicode"

	// Test with unicode service name
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_VeryLongServiceName(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	serviceName := string(make([]byte, 10000)) // Very long service name

	// Test with very long service name
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_VeryLongServiceName(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	timeout := 100 * time.Millisecond
	serviceName := string(make([]byte, 10000)) // Very long service name

	// Test with very long service name
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestService_Ping_ServiceNameWithNewlines(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	serviceName := "test\nservice\nwith\nnewlines"

	// Test with service name containing newlines
	assert.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestService_PingWithTimeout_ServiceNameWithNewlines(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	timeout := 100 * time.Millisecond
	serviceName := "test\nservice\nwith\nnewlines"

	// Test with service name containing newlines
	assert.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}
