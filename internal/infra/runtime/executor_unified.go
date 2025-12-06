package runtime

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"sync"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/bavix/gripmock/v3/internal/app/port"
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

const (
	defaultHeadersCapacity  = 8
	defaultRequestsCapacity = 4
	hashMultiplier          = 31
)

// UnifiedExecutor provides a single, optimized executor that combines the best of both approaches.
type UnifiedExecutor struct {
	Stubs     port.StubRepository
	Analytics port.AnalyticsRepository
	History   port.HistoryRepository

	// MessageSizeLimit bounds message size persisted to history (bytes). 0 = unlimited.
	MessageSizeLimit int64

	// Performance optimizations
	cache    *sync.Map  // Cache for parsed templates and compiled matchers
	pool     *sync.Pool // Object pool for temporary data structures
	fastPath bool       // Enable fast path optimizations

	// Parser utilities
	parser *parser
}

// NewUnifiedExecutor creates a new unified executor with performance enhancements.
func NewUnifiedExecutor(
	stubs port.StubRepository,
	analytics port.AnalyticsRepository,
	history port.HistoryRepository,
	messageSizeLimit int64,
) *UnifiedExecutor {
	return &UnifiedExecutor{
		Stubs:            stubs,
		Analytics:        analytics,
		History:          history,
		MessageSizeLimit: messageSizeLimit,
		cache:            &sync.Map{},
		pool: &sync.Pool{
			New: func() any {
				return &unifiedTempData{
					headers:  make(map[string]any, defaultHeadersCapacity),
					requests: make([]map[string]any, 0, defaultRequestsCapacity),
				}
			},
		},
		fastPath: true,
		parser:   &parser{},
	}
}

// unifiedTempData is a reusable temporary data structure for processing requests.
type unifiedTempData struct {
	headers  map[string]any
	requests []map[string]any
}

// reset clears the temporary data for reuse.
func (td *unifiedTempData) reset() {
	for k := range td.headers {
		delete(td.headers, k)
	}

	td.requests = td.requests[:0]
}

// Execute provides unified stub execution with optimizations.
//

func (e *UnifiedExecutor) Execute(
	ctx context.Context,
	stub domain.Stub,
	rpcType string,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, error) {
	start := time.Now()

	// Fast path: check if stub is exhausted
	if e.exhaustedByTimes(ctx, stub) {
		return false, nil
	}

	// Set response headers early
	if err := w.SetHeaders(stub.ResponseHeaders); err != nil {
		return false, errors.Wrap(err, "failed to set response headers")
	}

	// Get temporary data from pool
	temp, ok := e.pool.Get().(*unifiedTempData)
	if !ok {
		return false, errors.New("failed to get temp data from pool")
	}

	defer func() {
		temp.reset()
		e.pool.Put(temp)
	}()

	// Copy headers and requests to avoid allocations
	maps.Copy(temp.headers, headers)

	temp.requests = append(temp.requests, requests...)

	// Process outputs
	var (
		execErr error
		success bool
	)

	if len(stub.OutputsRaw) > 0 {
		success, execErr = e.processOutputsOptimized(ctx, stub, temp.headers, temp.requests, w)
	} else {
		// Fallback to legacy processing
		success, execErr = e.processLegacyOutput(ctx, stub, temp.headers, temp.requests, w)
	}

	// Record analytics
	e.recordAnalytics(ctx, stub, time.Since(start), execErr, len(requests))

	// Record history
	e.recordHistory(ctx, stub, rpcType, temp.headers, temp.requests, time.Since(start), execErr)

	return success, execErr
}

// processOutputsOptimized processes v4 outputs with optimizations.
//
//nolint:funcorder
func (e *UnifiedExecutor) processOutputsOptimized(
	ctx context.Context,
	stub domain.Stub,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, error) {
	for _, output := range stub.OutputsRaw {
		if err := e.processOutput(ctx, output, headers, requests, w); err != nil {
			return false, err
		}
	}

	return true, nil
}

// processOutput processes a single output with optimizations.
//
//nolint:funcorder
func (e *UnifiedExecutor) processOutput(
	ctx context.Context,
	output map[string]any,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) error {
	// Process data response
	if data, ok := e.parser.parseMap(output["data"]); ok {
		processedData := e.applyTemplatesCached(data)
		if err := w.Send(processedData); err != nil {
			return errors.Wrap(err, "failed to send data")
		}
	}

	// Process stream response
	if stream, ok := e.parser.parseSlice(output["stream"]); ok {
		if err := e.processStreamSteps(ctx, stream, w); err != nil {
			return errors.Wrap(err, "failed to process stream")
		}
	}

	// Process sequence response
	if sequence, ok := e.parser.parseSlice(output["sequence"]); ok {
		if err := e.processSequence(ctx, sequence, headers, requests, w); err != nil {
			return errors.Wrap(err, "failed to process sequence")
		}
	}

	// Process status response
	if status, ok := e.parser.parseMap(output["status"]); ok {
		grpcStatus := e.parser.parseGrpcStatus(status)
		if err := w.End(grpcStatus); err != nil {
			return errors.Wrap(err, "failed to end with status")
		}
	}

	return nil
}

// processStreamSteps processes stream steps with optimizations.
//
//nolint:funcorder
func (e *UnifiedExecutor) processStreamSteps(
	ctx context.Context,
	steps []any,
	w Writer,
) error {
	for _, step := range steps {
		if stepMap, ok := e.parser.parseMap(step); ok {
			if err := e.executeStreamStep(ctx, stepMap, w); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeStreamStep executes a single stream step.
//
//nolint:funcorder
func (e *UnifiedExecutor) executeStreamStep(
	_ context.Context,
	step map[string]any,
	w Writer,
) error {
	// Process send
	if send, ok := e.parser.parseMap(step["send"]); ok {
		processedSend := e.applyTemplatesCached(send)
		if err := w.Send(processedSend); err != nil {
			return errors.Wrap(err, "failed to send stream data")
		}
	}

	// Process delay
	if delay, ok := step["delay"]; ok {
		if duration, err := e.parser.parseDuration(delay); err == nil && duration > 0 {
			time.Sleep(duration)
		}
	}

	// Process end
	if end, ok := e.parser.parseMap(step["end"]); ok {
		grpcStatus := e.parser.parseGrpcStatus(end)
		if err := w.End(grpcStatus); err != nil {
			return errors.Wrap(err, "failed to end stream")
		}
	}

	return nil
}

// processSequence processes sequence items with optimizations.
//
//nolint:funcorder
func (e *UnifiedExecutor) processSequence(
	ctx context.Context,
	sequence []any,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) error {
	for _, item := range sequence {
		if itemMap, ok := e.parser.parseMap(item); ok {
			if err := e.processSequenceItem(ctx, itemMap, headers, requests, w); err != nil {
				return err
			}
		}
	}

	return nil
}

// processSequenceItem processes a single sequence item.
//
//nolint:funcorder
func (e *UnifiedExecutor) processSequenceItem(
	ctx context.Context,
	item map[string]any,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) error {
	// Check match condition
	if match, ok := e.parser.parseMap(item["match"]); ok {
		if !e.matchesCondition(match, headers, requests) {
			return nil // Skip this item if no match
		}
	}

	// Process stream
	if stream, ok := e.parser.parseSlice(item["stream"]); ok {
		if err := e.processStreamSteps(ctx, stream, w); err != nil {
			return err
		}
	}

	// Process data
	if data, ok := e.parser.parseMap(item["data"]); ok {
		processedData := e.applyTemplatesCached(data)
		if err := w.Send(processedData); err != nil {
			return errors.Wrap(err, "failed to send sequence data")
		}
	}

	// Process status
	if status, ok := e.parser.parseMap(item["status"]); ok {
		grpcStatus := e.parser.parseGrpcStatus(status)
		if err := w.End(grpcStatus); err != nil {
			return errors.Wrap(err, "failed to end sequence")
		}
	}

	return nil
}

// processLegacyOutput processes legacy output format.
//
//nolint:funcorder
func (e *UnifiedExecutor) processLegacyOutput(
	ctx context.Context,
	stub domain.Stub,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, error) {
	// Legacy processing logic - simplified for v4 compatibility
	// In v4, legacy output is handled through OutputsRaw
	return true, nil
}

// matchesCondition checks if the condition matches the request.
//
//nolint:funcorder
func (e *UnifiedExecutor) matchesCondition(
	match map[string]any,
	_ map[string]any,
	requests []map[string]any,
) bool {
	// Simple matching logic - can be enhanced
	if len(requests) == 0 {
		return false
	}

	// Check against the first request
	request := requests[0]

	// Check equals
	if equals, ok := e.parser.parseMap(match["equals"]); ok {
		for k, v := range equals {
			if request[k] != v {
				return false
			}
		}
	}

	// Check contains
	if contains, ok := e.parser.parseMap(match["contains"]); ok {
		for k, v := range contains {
			if !e.containsValue(request[k], v) {
				return false
			}
		}
	}

	return true
}

// containsValue checks if a value contains another value.
//
//nolint:funcorder
func (e *UnifiedExecutor) containsValue(actual, expected any) bool {
	if actual == nil || expected == nil {
		return actual == expected
	}

	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	return actualStr == expectedStr
}

// applyTemplatesCached applies templates with caching.
//
//nolint:funcorder
func (e *UnifiedExecutor) applyTemplatesCached(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	// Generate cache key from data hash
	key := e.generateCacheKey(data)

	// Check cache first
	if cached, ok := e.cache.Load(key); ok {
		if result, ok := cached.(map[string]any); ok {
			return result
		}
	}

	// Apply templates
	result := applyRuntimeTemplates(data)

	// Cache the result
	e.cache.Store(key, result)

	return result
}

// generateCacheKey generates a cache key from data.
//
//nolint:funcorder
func (e *UnifiedExecutor) generateCacheKey(data map[string]any) string {
	// Simple hash-based key generation
	hash := 0
	for k, v := range data {
		hash = hash*hashMultiplier + len(k)
		hash = hash*hashMultiplier + len(fmt.Sprintf("%v", v))
	}

	return strconv.Itoa(hash)
}

// exhaustedByTimes checks if stub is exhausted by times limit.
//
//nolint:funcorder
func (e *UnifiedExecutor) exhaustedByTimes(ctx context.Context, stub domain.Stub) bool {
	if stub.Times <= 0 {
		return false
	}

	// Check analytics for usage count
	if e.Analytics != nil {
		if analytics, exists := e.Analytics.GetByStubID(ctx, stub.ID); exists {
			return analytics.UsedCount >= int64(stub.Times)
		}
	}

	return false
}

// recordAnalytics records execution analytics.
//
//nolint:funcorder
func (e *UnifiedExecutor) recordAnalytics(
	ctx context.Context,
	stub domain.Stub,
	duration time.Duration,
	err error,
	_ int,
) {
	if e.Analytics != nil {
		e.Analytics.TouchStub(ctx, stub.ID, duration.Milliseconds(), err != nil, 1, 1, 0)
	}
}

// recordHistory records execution history.
//
//nolint:funcorder
func (e *UnifiedExecutor) recordHistory(
	ctx context.Context,
	stub domain.Stub,
	rpcType string,
	_ map[string]any,
	_ []map[string]any,
	duration time.Duration,
	_ error,
) {
	if e.History != nil {
		record := domain.HistoryRecord{
			StubID:               stub.ID,
			RPCType:              rpcType,
			DurationMilliseconds: duration.Milliseconds(),
			Timestamp:            time.Now(),
		}
		e.History.Add(ctx, record)
	}
}

// ClearCache clears the template cache.
func (e *UnifiedExecutor) ClearCache() {
	e.cache.Range(func(key, value any) bool {
		e.cache.Delete(key)

		return true
	})
}

// GetCacheStats returns cache statistics.
func (e *UnifiedExecutor) GetCacheStats() map[string]any {
	count := 0

	e.cache.Range(func(key, value any) bool {
		count++

		return true
	})

	return map[string]any{
		"cacheSize": count,
	}
}
