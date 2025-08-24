package runtime

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// ExecutionHistoryEntry represents a single execution record.
type ExecutionHistoryEntry struct {
	Timestamp    time.Time        `json:"timestamp"`
	StubID       string           `json:"stubId"`
	Service      string           `json:"service"`
	Method       string           `json:"method"`
	RequestCount int              `json:"requestCount"`
	Headers      map[string]any   `json:"headers"`
	Requests     []map[string]any `json:"requests"`
	Success      bool             `json:"success"`
	Error        string           `json:"error,omitempty"`
	Duration     time.Duration    `json:"duration"`
}

// AnalyticsExecutor executes stubs with optional analytics and logging.
type AnalyticsExecutor struct {
	analytics *Analytics
	logging   bool
}

// NewAnalyticsExecutor creates a new executor with optional analytics and logging.
func NewAnalyticsExecutor(analytics *Analytics, logging bool) *AnalyticsExecutor {
	return &AnalyticsExecutor{
		analytics: analytics,
		logging:   logging,
	}
}

// Execute executes a stub with analytics and logging if enabled.
func (e *AnalyticsExecutor) Execute(
	ctx context.Context,
	stub domain.StubStrict,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, error) {
	start := time.Now()

	e.logBeforeExecution(ctx, stub, requests)

	// Record stub usage if analytics enabled
	if e.analytics != nil {
		e.analytics.RecordStubUsage(stub.ID, stub.Service, stub.Method)
	}

	// Execute the actual stub (delegate to V4ExecutorStrict)
	// This would be implemented by the actual executor
	result, err := e.executeStub(ctx, stub, headers, requests, w)

	e.recordAnalytics(ctx, stub, headers, requests, start, result, err)
	e.logAfterExecution(ctx, start, result, err)

	return result, err
}

// executeStub is a placeholder for the actual stub execution logic.
// In a real implementation, this would delegate to ExecutorStrict.
func (e *AnalyticsExecutor) executeStub(
	ctx context.Context,
	stub domain.StubStrict,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, error) {
	// This is a placeholder - in real implementation this would call the actual executor
	// For now, we'll just return success
	return true, nil
}

// logBeforeExecution logs information before stub execution.
func (e *AnalyticsExecutor) logBeforeExecution(ctx context.Context, stub domain.StubStrict, requests []map[string]any) {
	if !e.logging {
		return
	}

	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("service", stub.Service).
		Str("method", stub.Method).
		Str("stub_id", stub.ID).
		Msg("Executing stub")
	logger.Info().
		Int("request_count", len(requests)).
		Msg("Request count")
}

// recordAnalytics records execution analytics if enabled.
func (e *AnalyticsExecutor) recordAnalytics(
	_ context.Context,
	stub domain.StubStrict,
	headers map[string]any,
	requests []map[string]any,
	start time.Time,
	_ bool,
	err error,
) {
	if e.analytics == nil {
		return
	}

	duration := time.Since(start)
	success := err == nil
	e.analytics.RecordExecution(stub.Service, stub.Method, duration, success, len(requests))

	// Record metrics
	labels := map[string]string{
		"service": stub.Service,
		"method":  stub.Method,
		"stub_id": stub.ID,
	}
	e.analytics.IncrementCounter("stub_executions_total", labels)
	e.analytics.RecordHistogram("stub_execution_duration_seconds", duration.Seconds(), labels)

	// Add to history
	entry := ExecutionHistoryEntry{
		Timestamp:    time.Now(),
		StubID:       stub.ID,
		Service:      stub.Service,
		Method:       stub.Method,
		RequestCount: len(requests),
		Headers:      headers,
		Requests:     requests,
		Success:      err == nil,
		Duration:     duration,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	e.analytics.AddToHistory(entry)
}

// logAfterExecution logs information after stub execution.
func (e *AnalyticsExecutor) logAfterExecution(ctx context.Context, start time.Time, result bool, err error) {
	if !e.logging {
		return
	}

	duration := time.Since(start)

	logger := zerolog.Ctx(ctx)
	if err != nil {
		logger.Error().
			Dur("duration", duration).
			Err(err).
			Msg("Execution failed")
	} else {
		logger.Info().
			Dur("duration", duration).
			Bool("result", result).
			Msg("Execution completed")
	}
}
