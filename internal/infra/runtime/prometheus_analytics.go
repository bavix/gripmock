package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// PrometheusAnalytics provides analytics collection using Prometheus metrics.
type PrometheusAnalytics struct {
	mu sync.RWMutex

	// Prometheus registry and metrics
	registry          *prometheus.Registry
	executionCounter  *prometheus.CounterVec
	executionDuration *prometheus.HistogramVec
	executionSuccess  *prometheus.CounterVec
	executionFailure  *prometheus.CounterVec
	stubUsageCounter  *prometheus.CounterVec
	requestCounter    *prometheus.CounterVec
	responseSizeGauge *prometheus.GaugeVec
	activeConnections *prometheus.GaugeVec
	errorRate         *prometheus.GaugeVec

	// History tracking
	history    []domain.HistoryRecord
	maxHistory int
}

// NewPrometheusAnalytics creates a new Prometheus-based analytics collector.
func NewPrometheusAnalytics(maxHistory int) *PrometheusAnalytics {
	registry := prometheus.NewRegistry()

	pa := &PrometheusAnalytics{
		registry:   registry,
		history:    make([]domain.HistoryRecord, 0, maxHistory),
		maxHistory: maxHistory,
	}

	pa.initializeMetrics()
	pa.registerMetrics()

	return pa
}

// initializeMetrics creates all Prometheus metrics.
//
//nolint:funcorder
func (pa *PrometheusAnalytics) initializeMetrics() {
	pa.initExecutionMetrics()
	pa.initUsageMetrics()
	pa.initConnectionMetrics()
}

// initExecutionMetrics creates execution-related metrics.
//
//nolint:funcorder
func (pa *PrometheusAnalytics) initExecutionMetrics() {
	pa.executionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gripmock_executions_total",
			Help: "Total number of stub executions",
		},
		[]string{"service", "method", "stub_id"},
	)

	pa.executionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gripmock_execution_duration_seconds",
			Help:    "Duration of stub executions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "stub_id"},
	)

	pa.executionSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gripmock_executions_success_total",
			Help: "Total number of successful stub executions",
		},
		[]string{"service", "method", "stub_id"},
	)

	pa.executionFailure = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gripmock_executions_failure_total",
			Help: "Total number of failed stub executions",
		},
		[]string{"service", "method", "stub_id"},
	)
}

// initUsageMetrics creates usage-related metrics.
//
//nolint:funcorder
func (pa *PrometheusAnalytics) initUsageMetrics() {
	pa.stubUsageCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gripmock_stub_usage_total",
			Help: "Total number of times each stub was used",
		},
		[]string{"stub_id", "service", "method"},
	)

	pa.requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gripmock_requests_total",
			Help: "Total number of requests processed",
		},
		[]string{"service", "method", "rpc_type"},
	)

	pa.responseSizeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gripmock_response_size_bytes",
			Help: "Size of responses in bytes",
		},
		[]string{"service", "method", "stub_id"},
	)
}

// initConnectionMetrics creates connection-related metrics.
//
//nolint:funcorder
func (pa *PrometheusAnalytics) initConnectionMetrics() {
	pa.activeConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gripmock_active_connections",
			Help: "Number of active connections",
		},
		[]string{"service", "method"},
	)

	pa.errorRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gripmock_error_rate",
			Help: "Error rate as a percentage",
		},
		[]string{"service", "method"},
	)
}

// registerMetrics registers all metrics with the registry.
//
//nolint:funcorder
func (pa *PrometheusAnalytics) registerMetrics() {
	pa.registry.MustRegister(
		pa.executionCounter,
		pa.executionDuration,
		pa.executionSuccess,
		pa.executionFailure,
		pa.stubUsageCounter,
		pa.requestCounter,
		pa.responseSizeGauge,
		pa.activeConnections,
		pa.errorRate,
	)
}

// RecordExecution records execution statistics using Prometheus metrics.
func (pa *PrometheusAnalytics) RecordExecution(ctx context.Context, stubID string, duration time.Duration, hasError bool) {
	// Extract service and method from context or use defaults
	const (
		unknownService = "unknown"
		unknownMethod  = "unknown"
	)

	service := unknownService
	method := unknownMethod

	if stub, ok := pa.getStubFromContext(ctx); ok {
		service = stub.Service
		method = stub.Method
	}

	labels := prometheus.Labels{
		"service": service,
		"method":  method,
		"stub_id": stubID,
	}

	// Record execution counter
	pa.executionCounter.With(labels).Inc()

	// Record execution duration
	pa.executionDuration.With(labels).Observe(duration.Seconds())

	// Record success/failure
	if hasError {
		pa.executionFailure.With(labels).Inc()
	} else {
		pa.executionSuccess.With(labels).Inc()
	}

	// Update error rate
	pa.updateErrorRate(service, method)
}

// RecordStubUsage records stub usage using Prometheus metrics.
func (pa *PrometheusAnalytics) RecordStubUsage(ctx context.Context, stubID string) {
	const (
		unknownService = "unknown"
		unknownMethod  = "unknown"
	)

	service := unknownService
	method := unknownMethod

	if stub, ok := pa.getStubFromContext(ctx); ok {
		service = stub.Service
		method = stub.Method
	}

	labels := prometheus.Labels{
		"stub_id": stubID,
		"service": service,
		"method":  method,
	}

	pa.stubUsageCounter.With(labels).Inc()
}

// RecordRequest records request processing using Prometheus metrics.
func (pa *PrometheusAnalytics) RecordRequest(ctx context.Context, rpcType string, requestCount int) {
	const (
		unknownService = "unknown"
		unknownMethod  = "unknown"
	)

	service := unknownService
	method := unknownMethod

	if stub, ok := pa.getStubFromContext(ctx); ok {
		service = stub.Service
		method = stub.Method
	}

	labels := prometheus.Labels{
		"service":  service,
		"method":   method,
		"rpc_type": rpcType,
	}

	pa.requestCounter.With(labels).Add(float64(requestCount))
}

// RecordResponseSize records response size using Prometheus metrics.
func (pa *PrometheusAnalytics) RecordResponseSize(ctx context.Context, stubID string, sizeBytes int64) {
	const (
		unknownService = "unknown"
		unknownMethod  = "unknown"
	)

	service := unknownService
	method := unknownMethod

	if stub, ok := pa.getStubFromContext(ctx); ok {
		service = stub.Service
		method = stub.Method
	}

	labels := prometheus.Labels{
		"service": service,
		"method":  method,
		"stub_id": stubID,
	}

	pa.responseSizeGauge.With(labels).Set(float64(sizeBytes))
}

// SetActiveConnections sets the number of active connections.
func (pa *PrometheusAnalytics) SetActiveConnections(ctx context.Context, count int) {
	const (
		unknownService = "unknown"
		unknownMethod  = "unknown"
	)

	service := unknownService
	method := unknownMethod

	if stub, ok := pa.getStubFromContext(ctx); ok {
		service = stub.Service
		method = stub.Method
	}

	labels := prometheus.Labels{
		"service": service,
		"method":  method,
	}

	pa.activeConnections.With(labels).Set(float64(count))
}

// AddHistory adds a history record.
func (pa *PrometheusAnalytics) AddHistory(ctx context.Context, record domain.HistoryRecord) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// Add to history
	pa.history = append(pa.history, record)

	// Maintain max history size
	if len(pa.history) > pa.maxHistory {
		pa.history = pa.history[1:]
	}
}

// GetHistory returns all history records.
func (pa *PrometheusAnalytics) GetHistory(ctx context.Context) []domain.HistoryRecord {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]domain.HistoryRecord, len(pa.history))
	copy(result, pa.history)

	return result
}

// GetStubUsage returns usage count for a specific stub.
func (pa *PrometheusAnalytics) GetStubUsage(ctx context.Context, stubID string) (int64, bool) {
	// Note: This is a simplified approach. In a real implementation,
	// you might want to use a custom collector or cache the values
	// For now, return 0 as we don't have a way to get current counter values
	return 0, false
}

// GetExecutionStats returns execution statistics for a service/method combination.
func (pa *PrometheusAnalytics) GetExecutionStats(ctx context.Context, service, method string) (map[string]any, bool) {
	// This would require implementing a custom collector or using Prometheus query API
	// For now, return empty stats
	return make(map[string]any), false
}

// GetMetrics returns all current metrics as a map.
func (pa *PrometheusAnalytics) GetMetrics(ctx context.Context) map[string]any {
	// This would require implementing a custom collector or using Prometheus query API
	// For now, return empty metrics
	return make(map[string]any)
}

// Reset resets all metrics.
func (pa *PrometheusAnalytics) Reset(ctx context.Context) {
	// Reset Prometheus metrics
	pa.executionCounter.Reset()
	pa.executionDuration.Reset()
	pa.executionSuccess.Reset()
	pa.executionFailure.Reset()
	pa.stubUsageCounter.Reset()
	pa.requestCounter.Reset()
	pa.responseSizeGauge.Reset()
	pa.activeConnections.Reset()
	pa.errorRate.Reset()

	// Reset history
	pa.mu.Lock()
	pa.history = make([]domain.HistoryRecord, 0, pa.maxHistory)
	pa.mu.Unlock()
}

// Helper methods

// getStubFromContext extracts stub information from context.
func (pa *PrometheusAnalytics) getStubFromContext(ctx context.Context) (domain.Stub, bool) {
	// This is a placeholder. In a real implementation, you would store stub info in context
	return domain.Stub{}, false
}

// updateErrorRate updates the error rate metric.
func (pa *PrometheusAnalytics) updateErrorRate(service, method string) {
	labels := prometheus.Labels{
		"service": service,
		"method":  method,
	}

	// Note: This is a simplified approach. In a real implementation,
	// you would need to get the actual counter values and calculate the rate
	pa.errorRate.With(labels).Set(0.0)
}
