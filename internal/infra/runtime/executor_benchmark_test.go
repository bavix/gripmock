package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/runtime"
)

// getBenchStub returns benchmark stub for legacy executor.
func getBenchStub() types.Stub {
	return types.Stub{
		ID:      "bench-stub",
		Service: "bench-service",
		Method:  "bench-method",
		OutputsRaw: []map[string]any{
			{
				"data": map[string]any{
					"message": "Hello, World!",
					"count":   42,
					"active":  true,
				},
			},
		},
	}
}

// Mock writer for benchmarks.
type mockWriter struct {
	sent []map[string]any
}

func (m *mockWriter) SetHeaders(headers map[string]string) error { return nil }
func (m *mockWriter) Send(data map[string]any) error {
	m.sent = append(m.sent, data)

	return nil
}
func (m *mockWriter) SetTrailers(trailers map[string]string) error { return nil }
func (m *mockWriter) End(status *types.GrpcStatus) error           { return nil }

// Benchmark legacy executor.
func BenchmarkExecutor_Execute(b *testing.B) {
	executor := &runtime.Executor{}
	writer := &mockWriter{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = executor.Execute(
				context.Background(),
				getBenchStub(),
				"unary",
				nil,
				nil,
				writer,
			)
		}
	})
}

// Benchmark command execution.
func BenchmarkCommand_Execute(b *testing.B) {
	writer := &mockWriter{}
	sendCmd := runtime.NewSendCommand(
		map[string]any{"message": "Hello, World!"},
		nil,
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = sendCmd.Execute(context.Background(), writer)
		}
	})
}

// Benchmark delay command.
func BenchmarkDelayCommand_Execute(b *testing.B) {
	writer := &mockWriter{}
	delayCmd := runtime.NewDelayCommand(time.Microsecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = delayCmd.Execute(context.Background(), writer)
		}
	})
}

// Benchmark composite command.
func BenchmarkCompositeCommand_Execute(b *testing.B) {
	writer := &mockWriter{}
	commands := []runtime.Command{
		runtime.NewSendCommand(map[string]any{"message": "Hello"}, nil),
		runtime.NewDelayCommand(time.Microsecond),
		runtime.NewSendCommand(map[string]any{"message": "World"}, nil),
	}
	compositeCmd := runtime.NewCompositeCommand(commands...)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = compositeCmd.Execute(context.Background(), writer)
		}
	})
}
