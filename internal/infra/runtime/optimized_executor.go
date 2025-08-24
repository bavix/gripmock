package runtime

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/bavix/gripmock/v3/internal/app/port"
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/matcher"
)

// OptimizedExecutor provides high-performance stub execution with caching and optimizations.
type OptimizedExecutor struct {
	Stubs     port.StubRepository
	Analytics port.AnalyticsRepository
	History   port.HistoryRepository

	// MessageSizeLimit bounds message size persisted to history (bytes). 0 = unlimited.
	MessageSizeLimit int64

	// Performance optimizations
	cache    *sync.Map  // Cache for parsed templates and compiled matchers
	pool     *sync.Pool // Object pool for temporary data structures
	fastPath bool       // Enable fast path optimizations
}

// NewOptimizedExecutor creates a new optimized executor with performance enhancements.
func NewOptimizedExecutor(
	stubs port.StubRepository,
	analytics port.AnalyticsRepository,
	history port.HistoryRepository,
	messageSizeLimit int64,
) *OptimizedExecutor {
	return &OptimizedExecutor{
		Stubs:            stubs,
		Analytics:        analytics,
		History:          history,
		MessageSizeLimit: messageSizeLimit,
		cache:            &sync.Map{},
		pool: &sync.Pool{
			New: func() any {
				return &tempData{
					headers:  make(map[string]any, defaultHeadersCapacity),
					requests: make([]map[string]any, 0, defaultRequestsCapacity),
				}
			},
		},
		fastPath: true,
	}
}

// tempData is a reusable temporary data structure for processing requests.
type tempData struct {
	headers  map[string]any
	requests []map[string]any
}

// reset clears the temporary data for reuse.
func (td *tempData) reset() {
	for k := range td.headers {
		delete(td.headers, k)
	}

	td.requests = td.requests[:0]
}

// Execute optimizes stub execution with caching and fast paths.
//
//nolint:gocognit,cyclop,funlen
func (e *OptimizedExecutor) Execute(
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
	temp, ok := e.pool.Get().(*tempData)
	if !ok {
		return false, errors.New("failed to get temp data from pool")
	}

	defer func() {
		temp.reset()
		e.pool.Put(temp)
	}()

	// Copy headers and requests to avoid allocations
	for k, v := range headers {
		temp.headers[k] = v
	}

	temp.requests = append(temp.requests, requests...)

	var (
		used      bool
		sendCount int64
		dataCount int64
		endCount  int64
	)

	// Optimized output processing with early exits
	for _, output := range stub.OutputsRaw {
		// Fast path: data response (most common case)
		if data, ok := output["data"]; ok {
			if dataMap, ok := globalParser.parseMap(data); ok {
				// Apply templates with caching
				processedData := e.applyTemplatesCached(dataMap)
				if err := w.Send(processedData); err != nil {
					return false, err
				}

				used = true
				dataCount++

				break
			}
		}

		// Fast path: stream response
		if stream, ok := output["stream"]; ok {
			streamUsed, streamSendCount, err := e.processStreamOutput(ctx, stream, w)
			if err != nil {
				return false, err
			}

			if streamUsed {
				used = true
				sendCount += int64(streamSendCount)

				break
			}
		}

		// Backward compatibility: sequence logic (optimized)
		if seq, ok := pickSequenceRule(output); ok {
			u, sc, dc, ee, err := e.executeSequenceOptimized(ctx, seq, temp.headers, temp.requests, w)
			if err != nil {
				return false, err
			}

			if u {
				used = true
				sendCount += sc
				dataCount += dc
				endCount += ee

				break
			}
		}
	}

	// Finalize response
	if err := e.finalize(w, stub); err != nil {
		return used, err
	}

	// Update analytics and history
	e.touch(ctx, stub.ID, time.Since(start), false, sendCount, dataCount, endCount)
	e.history(ctx, stub, rpcType, requests, time.Since(start))

	return used, nil
}

// applyTemplatesCached applies runtime templates with caching for better performance.
//
//nolint:funcorder
func (e *OptimizedExecutor) applyTemplatesCached(data map[string]any) map[string]any {
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

// generateCacheKey creates a simple hash-based cache key for data.
//
//nolint:funcorder
func (e *OptimizedExecutor) generateCacheKey(data map[string]any) string {
	// Simple hash implementation for cache key
	var hash uint64
	for k, v := range data {
		hash = hash*hashMultiplier + uint64(len(k))
		hash = hash*hashMultiplier + uint64(fmt.Sprintf("%v", v)[0])
	}

	return strconv.FormatUint(hash, 16)
}

// executeStreamStepOptimized executes a stream step with optimizations.
//
//nolint:funcorder
func (e *OptimizedExecutor) executeStreamStepOptimized(_ context.Context, step map[string]any, w Writer) error {
	// Fast path: send step
	if send, ok := globalParser.parseMap(step["send"]); ok {
		processedData := e.applyTemplatesCached(send)

		return w.Send(processedData)
	}

	// Fast path: delay step
	if delay := globalParser.parseString(step["delay"]); delay != "" {
		if d, err := globalParser.parseDuration(delay); err == nil && d > 0 {
			time.Sleep(d)
		}
		// Send empty response if only delay is present
		return w.Send(map[string]any{})
	}

	// Fast path: end step
	if end, ok := globalParser.parseMap(step["end"]); ok {
		status := &domain.GrpcStatus{
			Code:    globalParser.parseString(end["code"]),
			Message: globalParser.parseString(end["message"]),
		}

		return w.End(status)
	}

	return nil
}

// executeSequenceOptimized executes sequence logic with optimizations.
//
//nolint:funcorder
func (e *OptimizedExecutor) executeSequenceOptimized(
	ctx context.Context,
	seq domain.SequenceRule,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, int64, int64, int64, error) {
	// Process sequence items
	for _, item := range seq.Sequence {
		// Check if item matches
		if item.Match != nil {
			if !matcher.Match(convertMatcher(*item.Match), firstOrEmpty(headers, requests)) {
				continue
			}
		}

		// Process stream steps
		if len(item.Stream) > 0 {
			return e.processStreamStepsOptimized(ctx, item.Stream, w)
		}

		// Process data response
		if len(item.Data) > 0 {
			processedData := e.applyTemplatesCached(item.Data)
			if err := w.Send(processedData); err != nil {
				return false, 0, 0, 0, err
			}

			return true, 0, 1, 0, nil
		}

		// Process status response
		if item.Status != nil {
			if err := w.End(item.Status); err != nil {
				return false, 0, 0, 0, err
			}

			return true, 0, 0, 1, nil
		}
	}

	return false, 0, 0, 0, nil
}

// processStreamStepsOptimized processes stream steps with optimizations.
//
//nolint:funcorder
func (e *OptimizedExecutor) processStreamStepsOptimized(
	ctx context.Context,
	steps []domain.StreamStep,
	w Writer,
) (bool, int64, int64, int64, error) {
	var (
		used      bool
		sendCount int64
		dataCount int64
		endCount  int64
	)

	for _, step := range steps {
		// Fast path: send step
		if len(step.Send) > 0 {
			processedData := e.applyTemplatesCached(step.Send)
			if err := w.Send(processedData); err != nil {
				return used, sendCount, dataCount, endCount, errors.Wrap(err, "failed to send sequence message")
			}

			sendCount++
			used = true

			continue
		}

		// Fast path: delay step
		if step.Delay != "" {
			if err := delay(ctx, step.Delay); err != nil {
				return used, sendCount, dataCount, endCount, err
			}

			continue
		}

		// Fast path: end step
		if step.End != nil {
			if err := w.End(step.End); err != nil {
				return used, sendCount, dataCount, endCount, errors.Wrap(err, "failed to end sequence")
			}

			endCount++
			used = true

			break
		}
	}

	return used, sendCount, dataCount, endCount, nil
}

// exhaustedByTimes checks if stub is exhausted with caching.
//
//nolint:funcorder
func (e *OptimizedExecutor) exhaustedByTimes(ctx context.Context, stub domain.Stub) bool {
	if stub.Times <= 0 || e.Analytics == nil {
		return false
	}

	if a, ok := e.Analytics.GetByStubID(ctx, stub.ID); ok {
		return int(a.UsedCount) >= stub.Times
	}

	return false
}

// finalize sets response trailers.
//
//nolint:funcorder
func (e *OptimizedExecutor) finalize(w Writer, stub domain.Stub) error {
	if err := w.SetTrailers(stub.ResponseTrailers); err != nil {
		return errors.Wrap(err, "failed to set response trailers")
	}

	return nil
}

// touch updates analytics with atomic operations.
//
//nolint:funcorder
func (e *OptimizedExecutor) touch(
	ctx context.Context,
	stubID string,
	duration time.Duration,
	hasError bool,
	sendCount, dataCount, endCount int64,
) {
	if e.Analytics == nil {
		return
	}

	// Use the correct method from AnalyticsRepository
	e.Analytics.TouchStub(ctx, stubID, duration.Milliseconds(), hasError, sendCount, dataCount, endCount)
}

// history records execution history with size limits.
//
//nolint:funcorder
func (e *OptimizedExecutor) history(
	ctx context.Context,
	stub domain.Stub,
	rpcType string,
	requests []map[string]any,
	duration time.Duration,
) {
	if e.History == nil {
		return
	}

	// Create history record
	record := domain.HistoryRecord{
		StubID:               stub.ID,
		RPCType:              rpcType,
		Requests:             requests,
		DurationMilliseconds: duration.Milliseconds(),
		Timestamp:            time.Now(),
	}

	e.History.Add(ctx, record)
}

// ClearCache clears the internal cache (useful for testing or memory management).
func (e *OptimizedExecutor) ClearCache() {
	e.cache = &sync.Map{}
}

// GetCacheStats returns cache statistics for monitoring.
func (e *OptimizedExecutor) GetCacheStats() int {
	size := 0

	e.cache.Range(func(_, _ any) bool {
		size++

		return true
	})

	return size
}

// processStreamOutput processes stream output with simplified logic.
func (e *OptimizedExecutor) processStreamOutput(ctx context.Context, stream any, w Writer) (bool, int, error) {
	streamArray, ok := globalParser.parseSlice(stream)
	if !ok {
		return false, 0, nil
	}

	sendCount := 0

	for _, step := range streamArray {
		stepMap, ok := globalParser.parseMap(step)
		if !ok {
			continue
		}

		if err := e.executeStreamStepOptimized(ctx, stepMap, w); err != nil {
			return false, 0, err
		}

		sendCount++
	}

	return true, sendCount, nil
}
