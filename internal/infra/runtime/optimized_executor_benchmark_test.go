package runtime_test

import (
	"context"
	"testing"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/runtime"
)

// getBenchStubOptimized returns benchmark stub for optimized executor.
func getBenchStubOptimized() domain.Stub {
	return domain.Stub{
		ID:      "bench-stub",
		Service: "bench-service",
		Method:  "bench-method",
		OutputsRaw: []map[string]any{
			{
				"data": map[string]any{
					"message": "Hello, World!",
					"count":   42,
					"active":  true,
					"nested": map[string]any{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
		},
	}
}

// getBenchStreamStub returns benchmark stream stub.
func getBenchStreamStub() domain.Stub {
	return domain.Stub{
		ID:      "bench-stream-stub",
		Service: "bench-service",
		Method:  "bench-method",
		OutputsRaw: []map[string]any{
			{
				"stream": []interface{}{
					map[string]any{
						"send": map[string]any{
							"message": "Hello",
							"count":   1,
						},
					},
					map[string]any{
						"delay": "1ms",
					},
					map[string]any{
						"send": map[string]any{
							"message": "World",
							"count":   2,
						},
					},
				},
			},
		},
	}
}

type benchmarkWriter struct {
	sent []map[string]any
}

func (b *benchmarkWriter) SetHeaders(headers map[string]string) error { return nil }
func (b *benchmarkWriter) Send(data map[string]any) error {
	b.sent = append(b.sent, data)

	return nil
}
func (b *benchmarkWriter) SetTrailers(trailers map[string]string) error { return nil }
func (b *benchmarkWriter) End(status *domain.GrpcStatus) error          { return nil }

// Benchmark legacy executor.
func BenchmarkLegacyExecutor_Execute(b *testing.B) {
	executor := &runtime.Executor{}
	writer := &benchmarkWriter{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = executor.Execute(
				context.Background(),
				getBenchStubOptimized(),
				"unary",
				nil,
				nil,
				writer,
			)
		}
	})
}

// Benchmark optimized executor.
func BenchmarkOptimizedExecutor_Execute(b *testing.B) {
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)
	writer := &benchmarkWriter{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = executor.Execute(
				context.Background(),
				getBenchStubOptimized(),
				"unary",
				nil,
				nil,
				writer,
			)
		}
	})
}

// Benchmark legacy executor with stream.
func BenchmarkLegacyExecutor_ExecuteStream(b *testing.B) {
	executor := &runtime.Executor{}
	writer := &benchmarkWriter{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = executor.Execute(
				context.Background(),
				getBenchStreamStub(),
				"server_stream",
				nil,
				nil,
				writer,
			)
		}
	})
}

// Benchmark optimized executor with stream.
func BenchmarkOptimizedExecutor_ExecuteStream(b *testing.B) {
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)
	writer := &benchmarkWriter{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = executor.Execute(
				context.Background(),
				getBenchStreamStub(),
				"server_stream",
				nil,
				nil,
				writer,
			)
		}
	})
}

// Benchmark template caching.
func BenchmarkOptimizedExecutor_TemplateCaching(b *testing.B) {
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)
	writer := &benchmarkWriter{}

	stubWithTemplates := domain.Stub{
		ID:      "template-stub",
		Service: "template-service",
		Method:  "template-method",
		OutputsRaw: []map[string]any{
			{
				"data": map[string]any{
					"message":   "Template message",
					"timestamp": "{{.timestamp}}",
					"random":    "{{.random}}",
					"nested": map[string]any{
						"key": "{{.nested.key}}",
					},
				},
			},
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = executor.Execute(
				context.Background(),
				stubWithTemplates,
				"unary",
				nil,
				nil,
				writer,
			)
		}
	})
}

// Benchmark object pool usage.
func BenchmarkOptimizedExecutor_ObjectPool(b *testing.B) {
	executor := runtime.NewOptimizedExecutor(nil, nil, nil, 0)
	writer := &benchmarkWriter{}

	// Test with headers and requests to exercise object pool
	headers := map[string]any{
		"content-type": "application/json",
		"user-agent":   "benchmark-test",
	}
	requests := []map[string]any{
		{"id": 1, "data": "request1"},
		{"id": 2, "data": "request2"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = executor.Execute(
				context.Background(),
				getBenchStubOptimized(),
				"unary",
				headers,
				requests,
				writer,
			)
		}
	})
}
