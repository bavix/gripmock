package runtime

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultHistogramCapacity is the default capacity for histogram slices.
	DefaultHistogramCapacity = 100
)

// Analytics provides in-memory analytics collection using atomics.
type Analytics struct {
	mu         sync.RWMutex
	executions map[string]*ExecutionStats
	stubUsage  map[string]*int64
	counters   map[string]*int64
	histograms map[string][]float64
	gauges     map[string]*int64
	history    []ExecutionHistoryEntry
	maxHistory int
}

// ExecutionStats holds statistics for service/method combinations using atomics.
type ExecutionStats struct {
	TotalExecutions int64 `json:"totalExecutions"`
	TotalDuration   int64 `json:"totalDuration"` // nanoseconds
	SuccessCount    int64 `json:"successCount"`
	FailureCount    int64 `json:"failureCount"`
	TotalRequests   int64 `json:"totalRequests"`
	LastExecution   int64 `json:"lastExecution"` // Unix timestamp
}

// NewAnalytics creates a new analytics collector.
func NewAnalytics(maxHistory int) *Analytics {
	return &Analytics{
		executions: make(map[string]*ExecutionStats),
		stubUsage:  make(map[string]*int64),
		counters:   make(map[string]*int64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]*int64),
		history:    make([]ExecutionHistoryEntry, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

// RecordExecution records execution statistics using atomics.
func (a *Analytics) RecordExecution(service, method string, duration time.Duration, success bool, requestCount int) {
	key := service + "." + method

	// Get or create stats atomically
	a.mu.Lock()

	stats, exists := a.executions[key]
	if !exists {
		stats = &ExecutionStats{}
		a.executions[key] = stats
	}

	a.mu.Unlock()

	// Update stats using atomics
	atomic.AddInt64(&stats.TotalExecutions, 1)
	atomic.AddInt64(&stats.TotalDuration, int64(duration))
	atomic.AddInt64(&stats.TotalRequests, int64(requestCount))
	atomic.StoreInt64(&stats.LastExecution, time.Now().Unix())

	if success {
		atomic.AddInt64(&stats.SuccessCount, 1)
	} else {
		atomic.AddInt64(&stats.FailureCount, 1)
	}
}

// RecordStubUsage records stub usage using atomics.
func (a *Analytics) RecordStubUsage(stubID, service, method string) {
	// Get or create counter atomically
	a.mu.Lock()

	counter, exists := a.stubUsage[stubID]
	if !exists {
		var initial int64

		counter = &initial
		a.stubUsage[stubID] = counter
	}

	a.mu.Unlock()

	// Increment counter using atomic
	atomic.AddInt64(counter, 1)
}

// IncrementCounter increments a counter using atomics.
func (a *Analytics) IncrementCounter(name string, labels map[string]string) {
	key := a.buildKey(name, labels)

	// Get or create counter atomically
	a.mu.Lock()

	counter, exists := a.counters[key]
	if !exists {
		var initial int64

		counter = &initial
		a.counters[key] = counter
	}

	a.mu.Unlock()

	// Increment using atomic
	atomic.AddInt64(counter, 1)
}

// RecordHistogram records a histogram value.
func (a *Analytics) RecordHistogram(name string, value float64, labels map[string]string) {
	key := a.buildKey(name, labels)

	// Get or create histogram atomically
	a.mu.Lock()

	histogram, exists := a.histograms[key]
	if !exists {
		histogram = make([]float64, 0, DefaultHistogramCapacity)
	}

	a.histograms[key] = append(histogram, value)
	a.mu.Unlock()
}

// RecordGauge records a gauge value using atomics.
func (a *Analytics) RecordGauge(name string, value float64, labels map[string]string) {
	key := a.buildKey(name, labels)

	// Get or create gauge atomically
	a.mu.Lock()

	gauge, exists := a.gauges[key]
	if !exists {
		var initial int64

		gauge = &initial
		a.gauges[key] = gauge
	}

	a.mu.Unlock()

	// Store using atomic (convert float64 to int64 for atomic storage)
	atomic.StoreInt64(gauge, int64(value))
}

// AddToHistory adds an entry to the history.
func (a *Analytics) AddToHistory(entry ExecutionHistoryEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add new entry
	a.history = append(a.history, entry)

	// Maintain max size
	if len(a.history) > a.maxHistory {
		a.history = a.history[1:]
	}
}

// GetStats returns execution statistics.
func (a *Analytics) GetStats() map[string]ExecutionStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]ExecutionStats)
	for k, v := range a.executions {
		// Read atomically
		result[k] = ExecutionStats{
			TotalExecutions: atomic.LoadInt64(&v.TotalExecutions),
			TotalDuration:   atomic.LoadInt64(&v.TotalDuration),
			SuccessCount:    atomic.LoadInt64(&v.SuccessCount),
			FailureCount:    atomic.LoadInt64(&v.FailureCount),
			TotalRequests:   atomic.LoadInt64(&v.TotalRequests),
			LastExecution:   atomic.LoadInt64(&v.LastExecution),
		}
	}

	return result
}

// GetStubUsage returns stub usage statistics.
func (a *Analytics) GetStubUsage() map[string]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]int)
	for k, v := range a.stubUsage {
		// Read atomically
		result[k] = int(atomic.LoadInt64(v))
	}

	return result
}

// GetCounters returns all counters.
func (a *Analytics) GetCounters() map[string]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]int)
	for k, v := range a.counters {
		// Read atomically
		result[k] = int(atomic.LoadInt64(v))
	}

	return result
}

// GetHistograms returns all histograms.
func (a *Analytics) GetHistograms() map[string][]float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string][]float64)
	for k, v := range a.histograms {
		result[k] = append([]float64{}, v...)
	}

	return result
}

// GetGauges returns all gauges.
func (a *Analytics) GetGauges() map[string]float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]float64)
	for k, v := range a.gauges {
		// Read atomically
		result[k] = float64(atomic.LoadInt64(v))
	}

	return result
}

// GetHistory returns history entries with limit.
func (a *Analytics) GetHistory(limit int) []ExecutionHistoryEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit <= 0 || limit > len(a.history) {
		limit = len(a.history)
	}

	result := make([]ExecutionHistoryEntry, limit)
	copy(result, a.history[len(a.history)-limit:])

	return result
}

// GetHistoryByStubID returns history entries for a specific stub.
func (a *Analytics) GetHistoryByStubID(stubID string, limit int) []ExecutionHistoryEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []ExecutionHistoryEntry
	for i := len(a.history) - 1; i >= 0 && len(result) < limit; i-- {
		if a.history[i].StubID == stubID {
			result = append([]ExecutionHistoryEntry{a.history[i]}, result...)
		}
	}

	return result
}

// GetHistoryByService returns history entries for a specific service.
func (a *Analytics) GetHistoryByService(service string, limit int) []ExecutionHistoryEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []ExecutionHistoryEntry
	for i := len(a.history) - 1; i >= 0 && len(result) < limit; i-- {
		if a.history[i].Service == service {
			result = append([]ExecutionHistoryEntry{a.history[i]}, result...)
		}
	}

	return result
}

// buildKey builds a key from name and labels.
func (a *Analytics) buildKey(name string, labels map[string]string) string {
	var b strings.Builder
	b.Grow(len(name) + len(labels)*4)
	b.WriteString(name)

	for k, v := range labels {
		_, _ = b.WriteString("_")
		_, _ = b.WriteString(k)
		_, _ = b.WriteString("_")
		_, _ = b.WriteString(v)
	}

	return b.String()
}
