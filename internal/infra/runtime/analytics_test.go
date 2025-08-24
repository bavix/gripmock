package runtime_test

import (
	"sync"
	"testing"
	"time"

	"github.com/bavix/gripmock/v3/internal/infra/runtime"
)

func TestAnalytics_AtomicOperations(t *testing.T) {
	t.Parallel()

	analytics := runtime.NewAnalytics(1000)

	// Test concurrent execution recording
	var wg sync.WaitGroup

	const (
		numGoroutines = 100
		numOperations = 10
	)

	for range numGoroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range numOperations {
				analytics.RecordExecution("test-service", "test-method", time.Millisecond, true, 1)
				analytics.RecordStubUsage("stub-1", "test-service", "test-method")
			}
		}()
	}

	wg.Wait()

	// Verify results
	stats := analytics.GetStats()
	if len(stats) != 1 {
		t.Fatalf("Expected 1 service/method combination, got %d", len(stats))
	}

	key := "test-service.test-method"
	stat := stats[key]
	expectedExecutions := int64(numGoroutines * numOperations)

	if stat.TotalExecutions != expectedExecutions {
		t.Errorf("Expected %d total executions, got %d", expectedExecutions, stat.TotalExecutions)
	}

	if stat.SuccessCount != expectedExecutions {
		t.Errorf("Expected %d success count, got %d", expectedExecutions, stat.SuccessCount)
	}

	if stat.FailureCount != 0 {
		t.Errorf("Expected 0 failure count, got %d", stat.FailureCount)
	}

	// Verify stub usage
	stubUsage := analytics.GetStubUsage()
	if len(stubUsage) != 1 {
		t.Fatalf("Expected 1 stub usage entry, got %d", len(stubUsage))
	}

	if stubUsage["stub-1"] != int(expectedExecutions) {
		t.Errorf("Expected stub-1 usage count %d, got %d", expectedExecutions, stubUsage["stub-1"])
	}
}

func TestAnalytics_MetricsOperations(t *testing.T) {
	t.Parallel()

	analytics := runtime.NewAnalytics(1000)

	// Test concurrent metrics recording
	var wg sync.WaitGroup

	const (
		numGoroutines = 50
		numOperations = 20
	)

	for i := range numGoroutines {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := range numOperations {
				analytics.IncrementCounter("test-counter", map[string]string{"label": "value"})
				analytics.RecordHistogram("test-histogram", float64(j), map[string]string{"label": "value"})
				analytics.RecordGauge("test-gauge", float64(id), map[string]string{"label": "value"})
			}
		}(i)
	}

	wg.Wait()

	// Verify counters
	counters := analytics.GetCounters()

	expectedCounterKey := "test-counter_label_value"
	if counters[expectedCounterKey] != numGoroutines*numOperations {
		t.Errorf("Expected counter value %d, got %d", numGoroutines*numOperations, counters[expectedCounterKey])
	}

	// Verify histograms
	histograms := analytics.GetHistograms()
	expectedHistogramKey := "test-histogram_label_value"

	histogramValues := histograms[expectedHistogramKey]
	if len(histogramValues) != numGoroutines*numOperations {
		t.Errorf("Expected %d histogram values, got %d", numGoroutines*numOperations, len(histogramValues))
	}

	// Verify gauges (gauge gets overwritten, so we can't predict the exact value)
	gauges := analytics.GetGauges()
	expectedGaugeKey := "test-gauge_label_value"

	gaugeValue := gauges[expectedGaugeKey]
	if gaugeValue < 0 || gaugeValue >= float64(numGoroutines) {
		t.Errorf("Expected gauge value to be between 0 and %d, got %f", numGoroutines, gaugeValue)
	}
}

func TestAnalytics_HistoryOperations(t *testing.T) {
	t.Parallel()

	analytics := runtime.NewAnalytics(1000)

	// Test concurrent history recording
	var wg sync.WaitGroup

	const (
		numGoroutines = 10
		numEntries    = 5
	)

	for range numGoroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range numEntries {
				entry := runtime.ExecutionHistoryEntry{
					Timestamp:    time.Now(),
					StubID:       "stub-1",
					Service:      "test-service",
					Method:       "test-method",
					RequestCount: 1,
					Headers:      map[string]any{},
					Requests:     []map[string]any{},
					Success:      true,
					Duration:     time.Millisecond,
				}
				analytics.AddToHistory(entry)
			}
		}()
	}

	wg.Wait()

	// Verify history size
	history := analytics.GetHistory(0) // Get all history
	expectedSize := numGoroutines * numEntries
	t.Logf("Total history entries: %d, expected: %d", len(history), expectedSize)

	if len(history) != expectedSize {
		t.Errorf("Expected %d history entries, got %d", expectedSize, len(history))
	}

	t.Logf("Analytics basic functionality works correctly")
}
