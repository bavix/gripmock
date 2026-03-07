package waiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testServiceName = "test-service"

// testContextKey is a custom type for context keys to avoid collisions.
type testContextKey string

const testKey testContextKey = "key"

func TestNewService(t *testing.T) {
	t.Parallel()

	// Test service creation with nil client
	service := NewService(nil)

	require.NotNil(t, service)
	require.Nil(t, service.client)
}

func TestServiceStruct(t *testing.T) {
	t.Parallel()

	// Test Service struct initialization
	service := &Service{
		client: nil,
	}
	require.NotNil(t, service)
	require.Nil(t, service.client)
}

func TestServicePIngWithTimeoutNilClient(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test ping with nil client (should panic or return error)
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngNilClient(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	serviceName := testServiceName

	// Test ping with nil client (should panic or return error)
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutContextTimeout(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Nanosecond)

	defer cancel()

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with already expired context
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngWithTimeoutZeroTimeout(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()

	timeout := time.Duration(0)
	serviceName := testServiceName

	// Test with zero timeout
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngWithTimeoutNegativeTimeout(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	timeout := -1 * time.Second
	serviceName := testServiceName

	// Test with negative timeout
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngEmptyServiceName(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()

	serviceName := ""

	// Test with empty service name
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngNilContext(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	var ctx context.Context = nil

	serviceName := testServiceName

	// Test with nil context
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutNilContext(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	var ctx context.Context = nil

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with nil context
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngWithTimeoutCancelledContext(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx, cancel := context.WithCancel(t.Context())

	cancel() // Cancel immediately

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with cancelled context
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngCancelledContext(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx, cancel := context.WithCancel(t.Context())

	cancel() // Cancel immediately

	serviceName := testServiceName

	// Test with cancelled context
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutVeryLongTimeout(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()

	timeout := 24 * time.Hour // Very long timeout
	serviceName := testServiceName

	// Test with very long timeout
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngWithTimeoutContextWithDeadline(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-1*time.Second))

	defer cancel()

	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with context that has already expired deadline
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngContextWithValues(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := context.WithValue(t.Context(), testKey, "value")
	serviceName := testServiceName

	// Test with context that has values
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutContextWithValues(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := context.WithValue(t.Context(), testKey, "value")
	timeout := 100 * time.Millisecond
	serviceName := testServiceName

	// Test with context that has values
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngServiceNameWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	serviceName := "test-service-with-special-chars-!@#$%^&*()"

	// Test with service name containing special characters
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutServiceNameWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	timeout := 100 * time.Millisecond
	serviceName := "test-service-with-special-chars-!@#$%^&*()"

	// Test with service name containing special characters
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngUnicodeServiceName(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	serviceName := "тест-сервис-с-unicode"

	// Test with unicode service name
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutUnicodeServiceName(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	timeout := 100 * time.Millisecond
	serviceName := "тест-сервис-с-unicode"

	// Test with unicode service name
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngVeryLongServiceName(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	serviceName := string(make([]byte, 10000)) // Very long service name

	// Test with very long service name
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutVeryLongServiceName(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	timeout := 100 * time.Millisecond
	serviceName := string(make([]byte, 10000)) // Very long service name

	// Test with very long service name
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}

func TestServicePIngServiceNameWithNewlines(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	serviceName := "test\nservice\nwith\nnewlines"

	// Test with service name containing newlines
	require.Panics(t, func() {
		_, _ = service.Ping(ctx, serviceName)
	})
}

func TestServicePIngWithTimeoutServiceNameWithNewlines(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := t.Context()
	timeout := 100 * time.Millisecond
	serviceName := "test\nservice\nwith\nnewlines"

	// Test with service name containing newlines
	require.Panics(t, func() {
		_, _ = service.PingWithTimeout(ctx, timeout, serviceName)
	})
}
