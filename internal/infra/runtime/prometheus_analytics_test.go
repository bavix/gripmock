package runtime_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/runtime"
)

func TestPrometheusAnalytics_BasicFunctionality(t *testing.T) {
	t.Parallel()
	// Setup
	analytics := runtime.NewPrometheusAnalytics(100)
	ctx := context.Background()

	// Test recording execution
	analytics.RecordExecution(ctx, "test-stub", 100*time.Millisecond, false)
	analytics.RecordExecution(ctx, "test-stub", 200*time.Millisecond, true)

	// Test recording stub usage
	analytics.RecordStubUsage(ctx, "test-stub")
	analytics.RecordStubUsage(ctx, "test-stub")

	// Test recording request
	analytics.RecordRequest(ctx, "unary", 5)

	// Test recording response size
	analytics.RecordResponseSize(ctx, "test-stub", 1024)

	// Test setting active connections
	analytics.SetActiveConnections(ctx, 10)

	// Test history
	record := domain.HistoryRecord{
		StubID:               "test-stub",
		RPCType:              "unary",
		DurationMilliseconds: 100,
		Timestamp:            time.Now(),
	}

	analytics.AddHistory(ctx, record)

	// Assertions
	history := analytics.GetHistory(ctx)
	assert.Len(t, history, 1)
	assert.Equal(t, "test-stub", history[0].StubID)
}

func TestPrometheusAnalytics_HistoryLimit(t *testing.T) {
	t.Parallel()
	// Setup with small history limit
	analytics := runtime.NewPrometheusAnalytics(3)
	ctx := context.Background()

	// Add more records than the limit
	for i := range 5 {
		record := domain.HistoryRecord{
			StubID:               "stub-" + string(rune('0'+i)),
			RPCType:              "unary",
			DurationMilliseconds: int64(i * 100),
			Timestamp:            time.Now(),
		}
		analytics.AddHistory(ctx, record)
	}

	// Assertions
	history := analytics.GetHistory(ctx)
	assert.Len(t, history, 3, "History should be limited to 3 records")

	// Should contain the last 3 records
	assert.Equal(t, "stub-2", history[0].StubID)
	assert.Equal(t, "stub-3", history[1].StubID)
	assert.Equal(t, "stub-4", history[2].StubID)
}

func TestPrometheusAnalytics_Reset(t *testing.T) {
	t.Parallel()
	// Setup
	analytics := runtime.NewPrometheusAnalytics(100)
	ctx := context.Background()

	// Add some data
	analytics.RecordExecution(ctx, "test-stub", 100*time.Millisecond, false)
	analytics.RecordStubUsage(ctx, "test-stub")

	record := domain.HistoryRecord{
		StubID:               "test-stub",
		RPCType:              "unary",
		DurationMilliseconds: 100,
		Timestamp:            time.Now(),
	}
	analytics.AddHistory(ctx, record)

	// Reset
	analytics.Reset(ctx)

	// Assertions
	history := analytics.GetHistory(ctx)
	assert.Empty(t, history, "History should be empty after reset")
}

func TestPrometheusAnalytics_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	// Setup
	analytics := runtime.NewPrometheusAnalytics(1000)
	ctx := context.Background()

	// Test concurrent access
	const (
		numGoroutines = 10
		numOperations = 100
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			for j := range numOperations {
				stubID := "stub-" + string(rune('0'+id))

				// Record execution
				analytics.RecordExecution(ctx, stubID, time.Duration(j)*time.Millisecond, j%2 == 0)

				// Record stub usage
				analytics.RecordStubUsage(ctx, stubID)

				// Record request
				analytics.RecordRequest(ctx, "unary", j)

				// Add history
				record := domain.HistoryRecord{
					StubID:               stubID,
					RPCType:              "unary",
					DurationMilliseconds: int64(j),
					Timestamp:            time.Now(),
				}
				analytics.AddHistory(ctx, record)
			}
		}(i)
	}

	wg.Wait()

	// Assertions
	history := analytics.GetHistory(ctx)
	assert.Len(t, history, 1000, "Should have 1000 history records")
}

func TestPrometheusAnalytics_Metrics(t *testing.T) {
	t.Parallel()
	// Setup
	analytics := runtime.NewPrometheusAnalytics(100)
	ctx := context.Background()

	// Add various metrics
	analytics.RecordExecution(ctx, "stub-1", 100*time.Millisecond, false)
	analytics.RecordExecution(ctx, "stub-1", 200*time.Millisecond, true)
	analytics.RecordExecution(ctx, "stub-2", 150*time.Millisecond, false)

	analytics.RecordStubUsage(ctx, "stub-1")
	analytics.RecordStubUsage(ctx, "stub-1")
	analytics.RecordStubUsage(ctx, "stub-2")

	analytics.RecordRequest(ctx, "unary", 5)
	analytics.RecordRequest(ctx, "server_stream", 3)

	analytics.RecordResponseSize(ctx, "stub-1", 1024)
	analytics.RecordResponseSize(ctx, "stub-2", 2048)

	analytics.SetActiveConnections(ctx, 10)

	// Test GetMetrics
	metrics := analytics.GetMetrics(ctx)
	assert.NotNil(t, metrics, "Metrics should not be nil")
}

func TestPrometheusAnalytics_GetStubUsage(t *testing.T) {
	t.Parallel()
	// Setup
	analytics := runtime.NewPrometheusAnalytics(100)
	ctx := context.Background()

	// Record some usage
	analytics.RecordStubUsage(ctx, "test-stub")
	analytics.RecordStubUsage(ctx, "test-stub")

	// Test GetStubUsage
	usage, exists := analytics.GetStubUsage(ctx, "test-stub")
	// Note: In current implementation, this returns 0, false
	assert.False(t, exists)
	assert.Equal(t, int64(0), usage)

	// Test non-existent stub
	usage, exists = analytics.GetStubUsage(ctx, "non-existent")
	assert.False(t, exists)
	assert.Equal(t, int64(0), usage)
}

func TestPrometheusAnalytics_GetExecutionStats(t *testing.T) {
	t.Parallel()
	// Setup
	analytics := runtime.NewPrometheusAnalytics(100)
	ctx := context.Background()

	// Record some executions
	analytics.RecordExecution(ctx, "test-stub", 100*time.Millisecond, false)
	analytics.RecordExecution(ctx, "test-stub", 200*time.Millisecond, true)

	// Test GetExecutionStats
	stats, exists := analytics.GetExecutionStats(ctx, "unknown", "unknown")
	// Note: In current implementation, this returns empty map, false
	assert.False(t, exists)
	assert.NotNil(t, stats)
}
