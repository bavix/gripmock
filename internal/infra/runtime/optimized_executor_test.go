package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/runtime"
)

type testWriter struct {
	sent []map[string]any
}

func (t *testWriter) SetHeaders(headers map[string]string) error { return nil }
func (t *testWriter) Send(data map[string]any) error {
	t.sent = append(t.sent, data)

	return nil
}
func (t *testWriter) SetTrailers(trailers map[string]string) error { return nil }
func (t *testWriter) End(status *domain.GrpcStatus) error          { return nil }

func TestOptimizedExecutor_BasicFunctionality(t *testing.T) {
	t.Parallel()
	// Setup
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)

	stub := domain.Stub{
		ID:      "test-stub",
		Service: "test-service",
		Method:  "test-method",
		OutputsRaw: []map[string]any{
			{
				"data": map[string]any{
					"message": "Hello, World!",
					"count":   42,
				},
			},
		},
	}

	writer := &testWriter{}

	// Execute
	used, err := executor.Execute(context.Background(), stub, "unary", nil, nil, writer)

	// Assertions
	require.NoError(t, err)
	assert.True(t, used)
	assert.Len(t, writer.sent, 1)
	assert.Equal(t, "Hello, World!", writer.sent[0]["message"])
	assert.Equal(t, 42, writer.sent[0]["count"])
}

func TestOptimizedExecutor_StreamResponse(t *testing.T) {
	t.Parallel()
	// Setup
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)

	stub := domain.Stub{
		ID:      "test-stub",
		Service: "test-service",
		Method:  "test-method",
		OutputsRaw: []map[string]any{
			{
				"stream": []interface{}{
					map[string]any{
						"send": map[string]any{
							"message": "Hello",
						},
					},
					map[string]any{
						"send": map[string]any{
							"message": "World",
						},
					},
				},
			},
		},
	}

	writer := &testWriter{}

	// Execute
	used, err := executor.Execute(context.Background(), stub, "server_stream", nil, nil, writer)

	// Assertions
	require.NoError(t, err)
	assert.True(t, used)
	assert.Len(t, writer.sent, 2)
	assert.Equal(t, "Hello", writer.sent[0]["message"])
	assert.Equal(t, "World", writer.sent[1]["message"])
}

func TestOptimizedExecutor_CacheOperations(t *testing.T) {
	t.Parallel()
	// Setup
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)

	// Test cache stats
	initialStats := executor.GetCacheStats()
	assert.Equal(t, 0, initialStats)

	// Test cache clearing
	executor.ClearCache()
	statsAfterClear := executor.GetCacheStats()
	assert.Equal(t, 0, statsAfterClear)
}

func TestOptimizedExecutor_Performance(t *testing.T) {
	t.Parallel()
	// Setup
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)

	stub := domain.Stub{
		ID:      "perf-stub",
		Service: "perf-service",
		Method:  "perf-method",
		OutputsRaw: []map[string]any{
			{
				"data": map[string]any{
					"message": "Performance test",
					"count":   100,
					"active":  true,
				},
			},
		},
	}

	writer := &testWriter{}

	// Performance test
	start := time.Now()

	for range 1000 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, nil, writer)
		require.NoError(t, err)
		assert.True(t, used)
	}

	duration := time.Since(start)

	// Assertions
	assert.Len(t, writer.sent, 1000)
	assert.Less(t, duration, 5*time.Second, "Performance test should complete within 5 seconds")
}

func TestOptimizedExecutor_ObjectPool(t *testing.T) {
	t.Parallel()
	// Setup
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)

	stub := domain.Stub{
		ID:      "pool-stub",
		Service: "pool-service",
		Method:  "pool-method",
		OutputsRaw: []map[string]any{
			{
				"data": map[string]any{
					"message": "Pool test",
				},
			},
		},
	}

	writer := &testWriter{}

	// Test object pool reuse
	for range 100 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, nil, writer)
		require.NoError(t, err)
		assert.True(t, used)
	}

	// Assertions
	assert.Len(t, writer.sent, 100)
}

func TestOptimizedExecutor_TemplateCaching(t *testing.T) {
	t.Parallel()
	// Setup
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)

	stub := domain.Stub{
		ID:      "cache-stub",
		Service: "cache-service",
		Method:  "cache-method",
		OutputsRaw: []map[string]any{
			{
				"data": map[string]any{
					"message":   "Cached template",
					"timestamp": "{{.timestamp}}",
				},
			},
		},
	}

	writer := &testWriter{}

	// Execute multiple times to test caching
	for range 10 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, nil, writer)
		require.NoError(t, err)
		assert.True(t, used)
	}

	// Assertions
	assert.Len(t, writer.sent, 10)

	// Check cache stats
	cacheStats := executor.GetCacheStats()
	assert.Positive(t, cacheStats, "Cache should contain some entries")
}
