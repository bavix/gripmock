package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/bavix/gripmock/v3/internal/app/port"
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/matcher"
)

// Modern parsing utilities for better performance.
type parser struct{}

// parseString safely extracts string value with type assertion.
func (p *parser) parseString(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}
	// Handle other types by converting to string
	return fmt.Sprintf("%v", v)
}

// parseMap safely extracts map[string]any with type assertion.
func (p *parser) parseMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}

	if m, ok := v.(map[string]any); ok {
		return m, true
	}

	return nil, false
}

// parseSlice safely extracts []any with type assertion.
func (p *parser) parseSlice(v any) ([]any, bool) {
	if v == nil {
		return nil, false
	}

	if s, ok := v.([]any); ok {
		return s, true
	}

	return nil, false
}

// parseGrpcStatus creates GrpcStatus from map efficiently.
func (p *parser) parseGrpcStatus(m map[string]any) *domain.GrpcStatus {
	if m == nil {
		return nil
	}

	status := &domain.GrpcStatus{}
	if code, ok := m["code"]; ok {
		status.Code = p.parseString(code)
	}

	if message, ok := m["message"]; ok {
		status.Message = p.parseString(message)
	}

	return status
}

// parseDuration safely parses duration string.
func (p *parser) parseDuration(v any) (time.Duration, error) {
	if v == nil {
		return 0, nil
	}

	durationStr := p.parseString(v)
	if durationStr == "" {
		return 0, nil
	}

	return time.ParseDuration(durationStr)
}

// Global parser instance for reuse.
//
//nolint:gochecknoglobals
var globalParser = &parser{}

// Writer abstracts sending messages/metadata in the underlying transport (e.g., gRPC).
// Implementations must be side-effect free when no messages are sent.
type Writer interface {
	SetHeaders(headers map[string]string) error
	Send(message map[string]any) error
	SetTrailers(trailers map[string]string) error
	End(status *domain.GrpcStatus) error
}

// Executor executes v4 stub outputs and updates analytics/history.
type Executor struct {
	Stubs     port.StubRepository
	Analytics port.AnalyticsRepository
	History   port.HistoryRepository

	// MessageSizeLimit bounds message size persisted to history (bytes). 0 = unlimited.
	MessageSizeLimit int64
}

// Execute selects a matching output and executes it according to RPC type.
// It returns true when any effect occurred (send/data/end), which should mark the stub as used.
// Inputs are request headers and a slice of request payloads (for streaming).
//
//nolint:gocognit,cyclop,funlen
func (e *Executor) Execute(
	ctx context.Context,
	stub domain.Stub,
	rpcType string, // "unary" | "server_stream" | "client_stream" | "bidi"
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, error) {
	start := time.Now()

	var (
		used      bool
		sendCount int64
		dataCount int64
		endCount  int64
	)

	if e.exhaustedByTimes(ctx, stub) {
		return false, nil
	}

	if err := w.SetHeaders(stub.ResponseHeaders); err != nil {
		return false, errors.Wrap(err, "failed to set response headers")
	}

	// Modern optimized logic: check outputs directly with better performance
	for _, output := range stub.OutputsRaw {
		// Fast path: data response (unary/client streaming)
		if data, ok := output["data"]; ok {
			if dataMap, ok := globalParser.parseMap(data); ok {
				if err := w.Send(applyRuntimeTemplates(dataMap)); err != nil {
					return false, err
				}

				used = true
				dataCount++

				break
			}
		}

		//nolint:nestif
		// Fast path: stream response (server/bidirectional streaming)
		if stream, ok := output["stream"]; ok {
			if streamArray, ok := globalParser.parseSlice(stream); ok {
				for _, step := range streamArray {
					if stepMap, ok := globalParser.parseMap(step); ok {
						if err := e.executeStreamStepOptimized(ctx, stepMap, w); err != nil {
							return false, err
						}

						sendCount++
					}
				}

				used = true

				break
			}
		}

		// Backward compatibility: sequence logic
		if seq, ok := pickSequenceRule(output); ok {
			u, sc, dc, ee, err := e.executeSequence(ctx, seq, headers, requests, w)
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

	if err := e.finalize(w, stub); err != nil {
		return used, err
	}

	e.touch(ctx, stub.ID, time.Since(start), false, sendCount, dataCount, endCount)
	e.history(ctx, stub, rpcType, requests, time.Since(start))

	return used, nil
}

func (e *Executor) exhaustedByTimes(ctx context.Context, stub domain.Stub) bool {
	if stub.Times <= 0 || e.Analytics == nil {
		return false
	}

	if a, ok := e.Analytics.GetByStubID(ctx, stub.ID); ok {
		if int(a.UsedCount) >= stub.Times {
			return true
		}
	}

	return false
}

func (e *Executor) finalize(w Writer, stub domain.Stub) error {
	if err := w.SetTrailers(stub.ResponseTrailers); err != nil {
		return errors.Wrap(err, "failed to set response trailers")
	}

	return nil
}

func (e *Executor) executeStream(
	ctx context.Context,
	rule domain.StreamRule,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, int64, int64, error) {
	if rule.Match != nil && !matcher.Match(convertMatcher(*rule.Match), firstOrEmpty(headers, requests)) {
		return false, 0, 0, nil
	}

	u, sc, ee, err := e.processStreamSteps(ctx, rule.Stream, w)
	if err != nil {
		return u, sc, ee, err
	}

	used, sendCount, endCount := u, sc, ee

	u2, sc2, err := e.processSendEach(ctx, rule.SendEach, requests, w)
	if err != nil {
		return used, sendCount, endCount, err
	}

	return used || u2, sendCount + sc2, endCount, nil
}

func (e *Executor) processStreamSteps(
	ctx context.Context,
	steps []domain.StreamStep,
	w Writer,
) (bool, int64, int64, error) {
	var (
		used      bool
		sendCount int64
		endCount  int64
	)

	for _, step := range steps {
		if len(step.Send) > 0 {
			if err := w.Send(applyRuntimeTemplates(step.Send)); err != nil {
				return used, sendCount, endCount, errors.Wrap(err, "failed to send stream message")
			}

			sendCount++
			used = true

			continue
		}

		if step.Delay != "" {
			if err := delay(ctx, step.Delay); err != nil {
				return used, sendCount, endCount, err
			}

			continue
		}

		if step.End != nil {
			if err := w.End(step.End); err != nil {
				return used, sendCount, endCount, errors.Wrap(err, "failed to end stream")
			}

			endCount++
			used = true

			break
		}
	}

	return used, sendCount, endCount, nil
}

func (e *Executor) processSendEach(
	ctx context.Context,
	se *domain.SendEach,
	requests []map[string]any,
	w Writer,
) (bool, int64, error) {
	if se == nil {
		return false, 0, nil
	}

	var (
		used      bool
		sendCount int64
	)

	for range requests {
		if err := w.Send(applyRuntimeTemplates(se.Message)); err != nil {
			return used, sendCount, errors.Wrap(err, "failed to send sendEach message")
		}

		sendCount++
		used = true

		if se.Delay != "" {
			if err := delay(ctx, se.Delay); err != nil {
				return used, sendCount, err
			}
		}
	}

	return used, sendCount, nil
}

func (e *Executor) executeSequence(
	ctx context.Context,
	rule domain.SequenceRule,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, int64, int64, int64, error) {
	var (
		used      bool
		sendCount int64
		dataCount int64
		endCount  int64
	)

	// Simplified logic: just execute sequence elements sequentially
	// without complex matching logic
	for _, item := range rule.Sequence {
		u, sc, dc, ee, err := e.execSequenceItem(ctx, item, headers, requests, w)
		if err != nil {
			return used, sendCount, dataCount, endCount, err
		}

		used = used || u
		sendCount += sc
		dataCount += dc
		endCount += ee
	}

	return used, sendCount, dataCount, endCount, nil
}

func (e *Executor) execSequenceItem(
	ctx context.Context,
	item domain.SequenceItem,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, int64, int64, int64, error) {
	u, sc, ee, err := e.executeStream(ctx, domain.StreamRule{Stream: item.Stream, SendEach: item.SendEach}, headers, requests, w)
	if err != nil {
		return false, 0, 0, 0, err
	}

	used := u
	sendCount := sc
	endCount := ee

	if len(item.Data) > 0 {
		if err := w.Send(applyRuntimeTemplates(item.Data)); err != nil {
			return used, sendCount, 0, endCount, errors.Wrap(err, "failed to send data in sequence")
		}

		sendCount++
		dataCount := int64(1)
		used = true

		if item.Status != nil {
			if err := w.End(item.Status); err != nil {
				return used, sendCount, dataCount, endCount, errors.Wrap(err, "failed to end sequence")
			}

			endCount++
		}

		return used, sendCount, dataCount, endCount, nil
	}

	return used, sendCount, 0, endCount, nil
}

func (e *Executor) touch(ctx context.Context, stubID string, d time.Duration, wasErr bool, sendMsgs, dataRes, endEvents int64) {
	if e.Analytics == nil {
		return
	}

	e.Analytics.TouchStub(ctx, stubID, d.Milliseconds(), wasErr, sendMsgs, dataRes, endEvents)
}

func (e *Executor) history(
	ctx context.Context,
	stub domain.Stub,
	rpcType string,
	requests []map[string]any,
	d time.Duration,
) {
	if e.History == nil {
		return
	}

	rec := domain.HistoryRecord{
		Service:              stub.Service,
		Method:               stub.Method,
		RPCType:              rpcType,
		StubID:               stub.ID,
		FormatVersion:        "v4",
		RuleKind:             "auto",
		Requests:             requests,
		Responses:            nil,
		DurationMilliseconds: d.Milliseconds(),
	}
	e.History.Add(ctx, rec)
}

// --- helpers ---

func firstOrEmpty(headers map[string]any, requests []map[string]any) map[string]any {
	if len(requests) > 0 {
		return requests[0]
	}

	return headers
}

func parseStreamSteps(arr []any) []domain.StreamStep {
	steps := make([]domain.StreamStep, 0, len(arr))
	for _, s := range arr {
		sm, ok := globalParser.parseMap(s)
		if !ok {
			continue
		}

		var step domain.StreamStep
		if send, ok := globalParser.parseMap(sm["send"]); ok {
			step.Send = send
		}

		if d := globalParser.parseString(sm["delay"]); d != "" {
			step.Delay = d
		}

		if end, ok := globalParser.parseMap(sm["end"]); ok {
			step.End = globalParser.parseGrpcStatus(end)
		}

		steps = append(steps, step)
	}

	return steps
}

func parseSendEach(se map[string]any) *domain.SendEach {
	out := &domain.SendEach{
		Items: globalParser.parseString(se["items"]),
		As:    globalParser.parseString(se["as"]),
	}

	if msg, ok := globalParser.parseMap(se["message"]); ok {
		out.Message = msg
	}

	if d := globalParser.parseString(se["delay"]); d != "" {
		out.Delay = d
	}

	return out
}

func pickSequenceRule(m map[string]any) (domain.SequenceRule, bool) {
	arr, ok := m["sequence"].([]any)
	if !ok {
		return domain.SequenceRule{}, false
	}

	var rule domain.SequenceRule

	for _, it := range arr {
		if sm, ok := it.(map[string]any); ok {
			item := parseSequenceItem(sm)
			rule.Sequence = append(rule.Sequence, item)
		}
	}

	return rule, true
}

func parseSequenceItem(sm map[string]any) domain.SequenceItem {
	var item domain.SequenceItem

	if v, ok := globalParser.parseMap(sm["match"]); ok {
		item.Match = &domain.Matcher{Equals: v}
	}

	if data, ok := globalParser.parseMap(sm["data"]); ok {
		item.Data = data
	}

	if st, ok := globalParser.parseMap(sm["status"]); ok {
		item.Status = globalParser.parseGrpcStatus(st)
	}

	if arr2, ok := globalParser.parseSlice(sm["stream"]); ok {
		item.Stream = parseStreamSteps(arr2)
	}

	if se, ok := globalParser.parseMap(sm["sendEach"]); ok {
		item.SendEach = parseSendEach(se)
	}

	return item
}

// convertMatcher maps domain.Matcher to infra.matcher.Matcher.
func convertMatcher(dm domain.Matcher) matcher.Matcher {
	out := matcher.Matcher{
		Equals:           map[string]any{},
		Contains:         map[string]any{},
		Matches:          map[string]string{},
		IgnoreArrayOrder: dm.IgnoreArrayOrder,
	}
	if dm.Equals != nil {
		out.Equals = dm.Equals
	}

	if dm.Contains != nil {
		out.Contains = dm.Contains
	}

	if dm.Matches != nil {
		out.Matches = dm.Matches
	}

	if len(dm.Any) > 0 {
		out.Any = make([]matcher.Matcher, 0, len(dm.Any))
		for _, child := range dm.Any {
			out.Any = append(out.Any, convertMatcher(child))
		}
	}

	return out
}

// applyRuntimeTemplates walks a map and applies runtime transforms for string markers.
// Minimal implementation: returns input as-is. Placeholder for future expansion.
func applyRuntimeTemplates(m map[string]any) map[string]any { return m }

// delay parses simple durations like "100ms", "2s".
func delay(ctx context.Context, d string) error {
	if d == "" {
		return nil
	}

	dur, err := time.ParseDuration(d)
	if err != nil {
		return errors.Wrap(err, "invalid delay duration")
	}

	timer := time.NewTimer(dur)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// executeStreamStepOptimized is a high-performance version of executeStreamStep.
//
//nolint:cyclop,unparam
func (e *Executor) executeStreamStepOptimized(ctx context.Context, step map[string]any, w Writer) error {
	hasSend := false
	hasEnd := false

	// Handle send with optimized parsing
	if send, ok := step["send"]; ok {
		if sendMap, ok := globalParser.parseMap(send); ok {
			if err := w.Send(applyRuntimeTemplates(sendMap)); err != nil {
				return err
			}

			hasSend = true
		}
	}

	// Handle delay with optimized parsing
	if delay, ok := step["delay"]; ok {
		if d, err := globalParser.parseDuration(delay); err == nil && d > 0 {
			time.Sleep(d)
		}
	}

	// Handle end with optimized parsing
	if end, ok := step["end"]; ok {
		if endMap, ok := globalParser.parseMap(end); ok {
			status := globalParser.parseGrpcStatus(endMap)
			if err := w.End(status); err != nil {
				return err
			}

			hasEnd = true
		}
	}

	// If there's only delay without send/end, send empty response
	if !hasSend && !hasEnd {
		if err := w.Send(map[string]any{}); err != nil {
			return err
		}
	}

	return nil
}

// V4ExecutorStrict provides high-performance execution of strictly typed v4 stubs.
type V4ExecutorStrict struct {
	// Command factory for creating commands
	commandFactory *CommandFactory
}

// ExecuteStrict executes a strictly typed v4 stub with better performance.
//
//nolint:cyclop
func (e *V4ExecutorStrict) ExecuteStrict(
	ctx context.Context,
	stub domain.StubStrict,
	headers map[string]any,
	requests []map[string]any,
	w Writer,
) (bool, error) {
	var used bool

	// Modern optimized logic for strictly typed outputs
	for _, output := range stub.Outputs {
		// Fast path: data response (unary/client streaming)
		if output.Data != nil {
			if err := w.Send(applyRuntimeTemplates(output.Data.Content)); err != nil {
				return false, err
			}

			used = true

			break
		}

		// Fast path: stream response (server/bidirectional streaming)
		if len(output.Stream) > 0 {
			for _, step := range output.Stream {
				if err := e.executeStreamStepStrict(ctx, step, w); err != nil {
					return false, err
				}
			}

			used = true

			break
		}

		// Handle delay
		if output.Delay != nil {
			if d, err := globalParser.parseDuration(output.Delay.Duration); err == nil && d > 0 {
				time.Sleep(d)
			}
			// Send empty response if only delay is present
			if err := w.Send(map[string]any{}); err != nil {
				return false, err
			}

			used = true

			break
		}

		// Handle status
		if output.Status != nil {
			if err := w.End(output.Status); err != nil {
				return false, err
			}

			used = true

			break
		}
	}

	return used, nil
}

// executeStreamStepStrict executes a strictly typed stream step using Command pattern.
func (e *V4ExecutorStrict) executeStreamStepStrict(ctx context.Context, step domain.StreamStepStrict, w Writer) error {
	// Create commands from the step
	commands := e.commandFactory.CreateCommands(step)

	// Execute all commands
	for _, cmd := range commands {
		if err := cmd.Execute(ctx, w); err != nil {
			return err
		}
	}

	// If there's only delay without send/end, send empty response
	if step.Delay != nil && step.Send == nil && step.End == nil {
		if err := w.Send(map[string]any{}); err != nil {
			return err
		}
	}

	return nil
}
