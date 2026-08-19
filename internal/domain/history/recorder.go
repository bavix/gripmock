package history

import (
	"context"
	"encoding/json"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CallRecord represents a single gRPC call made to the mock.
type CallRecord struct {
	StubID          uuid.UUID         `json:"stubId,omitempty"`
	Service         string            `json:"service,omitempty"`
	Method          string            `json:"method,omitempty"`
	Session         string            `json:"session,omitempty"`
	Requests        []map[string]any  `json:"requests,omitempty"`
	Responses       []map[string]any  `json:"responses,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	Code            uint32            `json:"code,omitempty"`
	Error           string            `json:"error,omitempty"`
	ElapsedMS       int64             `json:"elapsedMs,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
}

// Recorder records gRPC calls for inspection and verification.
type Recorder interface {
	Record(call CallRecord)
}

// FilterOpts specifies filter criteria for recorded calls.
type FilterOpts struct {
	Service string
	Method  string
	Session string

	ErrorOnly bool
}

// Reader provides read access to recorded calls.
type Reader interface {
	All() []CallRecord
	Count() int
	Filter(opts FilterOpts) []CallRecord
	FilterByMethod(service, method string) []CallRecord
}

// SessionCleaner removes records for a specific session.
type SessionCleaner interface {
	DeleteSession(session string) int
}

// MemoryStore implements both Recorder and Reader (in-memory).
type storedRecord struct {
	call CallRecord
	size int64
}

type MemoryStore struct {
	mu              sync.RWMutex
	calls           []storedRecord
	limitBytes      int64
	messageMaxBytes int64
	redactKeys      map[string]struct{}
	currentBytes    int64
}

// MemoryStoreOption configures MemoryStore.
type MemoryStoreOption func(*MemoryStore)

// WithMessageMaxBytes limits Request/Response size; excess is replaced with truncation marker.
func WithMessageMaxBytes(n int64) MemoryStoreOption {
	return func(s *MemoryStore) {
		s.messageMaxBytes = n
	}
}

func WithRedactKeys(keys []string) MemoryStoreOption {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k != "" {
			m[strings.ToLower(k)] = struct{}{}
		}
	}

	return func(s *MemoryStore) {
		s.redactKeys = m
	}
}

// NewMemoryStore creates a store with optional byte limit.
func NewMemoryStore(limitBytes int64, opts ...MemoryStoreOption) *MemoryStore {
	s := &MemoryStore{limitBytes: limitBytes}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Record implements Recorder. The message maps are cloned, so the caller may
// keep mutating them; RecordOwned skips the clone when ownership is handed over.
func (s *MemoryStore) Record(call CallRecord) {
	if len(s.redactKeys) == 0 {
		call = cloneRecordMessages(call)
	}

	s.RecordOwned(call)
}

// RecordOwned stores the record without cloning its message maps: the caller
// hands them over and must not mutate them afterwards.
func (s *MemoryStore) RecordOwned(call CallRecord) {
	if len(s.redactKeys) > 0 {
		call = redactRecord(call, s.redactKeys)
	}

	if s.messageMaxBytes > 0 {
		call = truncateRecord(call, s.messageMaxBytes)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, storedRecord{call: call, size: estimateRecordSize(call)})
	s.currentBytes += s.calls[len(s.calls)-1].size

	for s.limitBytes > 0 && s.currentBytes > s.limitBytes && len(s.calls) > 1 {
		s.currentBytes -= s.calls[0].size
		s.calls = s.calls[1:]
	}
}

// Clear removes all recorded calls, resetting the store to empty in place.
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = nil
	s.currentBytes = 0
}

const redactedValue = "[REDACTED]"

func freshTruncatedMarker() map[string]any {
	return map[string]any{"_truncated": true}
}

func redactRecord(c CallRecord, keys map[string]struct{}) CallRecord {
	c.Requests = redactMaps(c.Requests, keys)
	c.Responses = redactMaps(c.Responses, keys)
	c.ResponseHeaders = redactStringMap(c.ResponseHeaders, keys)

	return c
}

func cloneRecordMessages(c CallRecord) CallRecord {
	c.Requests = cloneMaps(c.Requests)
	c.Responses = cloneMaps(c.Responses)
	c.ResponseHeaders = maps.Clone(c.ResponseHeaders)

	return c
}

func redactStringMap(m map[string]string, keys map[string]struct{}) map[string]string {
	if m == nil {
		return nil
	}

	out := make(map[string]string, len(m))

	for k, v := range m {
		if _, redact := keys[strings.ToLower(k)]; redact {
			out[k] = redactedValue
		} else {
			out[k] = v
		}
	}

	return out
}

func mapEach(ms []map[string]any, fn func(map[string]any) map[string]any) []map[string]any {
	if len(ms) == 0 {
		return ms
	}

	out := make([]map[string]any, len(ms))
	for i, m := range ms {
		out[i] = fn(m)
	}

	return out
}

func cloneMaps(ms []map[string]any) []map[string]any {
	return mapEach(ms, cloneMap)
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneValue(v)
	}

	return out
}

func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = cloneValue(e)
		}

		return out
	default:
		return t
	}
}

func redactMaps(ms []map[string]any, keys map[string]struct{}) []map[string]any {
	return mapEach(ms, func(m map[string]any) map[string]any { return redactMap(m, keys) })
}

func redactMap(m map[string]any, keys map[string]struct{}) map[string]any {
	if m == nil || len(keys) == 0 {
		return m
	}

	out := make(map[string]any, len(m))
	for k, v := range m {
		if _, ok := keys[strings.ToLower(k)]; ok {
			out[k] = redactedValue
		} else if sub, ok := asMap(v); ok {
			out[k] = redactMap(sub, keys)
		} else if arr := asSlice(v); arr != nil {
			out[k] = redactSlice(arr, keys)
		} else {
			out[k] = v
		}
	}

	return out
}

func redactSlice(arr []any, keys map[string]struct{}) []any {
	if arr == nil {
		return nil
	}

	out := make([]any, len(arr))
	for i, v := range arr {
		if sub, ok := asMap(v); ok {
			out[i] = redactMap(sub, keys)
		} else if subArr := asSlice(v); subArr != nil {
			out[i] = redactSlice(subArr, keys)
		} else {
			out[i] = v
		}
	}

	return out
}

func asMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}

	m, ok := v.(map[string]any)
	if ok {
		return m, true
	}

	return nil, false
}

func asSlice(v any) []any {
	if v == nil {
		return nil
	}

	arr, ok := v.([]any)
	if ok {
		return arr
	}

	return nil
}

func truncateRecord(c CallRecord, maxBytes int64) CallRecord {
	c.Requests = truncateMaps(c.Requests, maxBytes)
	c.Responses = truncateMaps(c.Responses, maxBytes)

	return c
}

func truncateMaps(ms []map[string]any, maxBytes int64) []map[string]any {
	return mapEach(ms, func(m map[string]any) map[string]any { return truncateMessage(m, maxBytes) })
}

func truncateMessage(m map[string]any, maxBytes int64) map[string]any {
	if b, err := json.Marshal(m); err == nil && int64(len(b)) > maxBytes {
		return freshTruncatedMarker()
	}

	return m
}

// estimateRecordSize approximates the JSON footprint without marshalling: the
// limit only needs proportional accounting, and json.Marshal on every record was
// the single most expensive step of Record.
func estimateRecordSize(c CallRecord) int64 {
	const recordOverhead = 160

	size := int64(recordOverhead)
	size += int64(len(c.Service) + len(c.Method) + len(c.Session) + len(c.Error))

	for _, m := range c.Requests {
		size += estimateMapSize(m)
	}

	for _, m := range c.Responses {
		size += estimateMapSize(m)
	}

	for k, v := range c.ResponseHeaders {
		size += int64(len(k) + len(v) + kvOverhead)
	}

	return size
}

const (
	kvOverhead      = 8
	scalarSize      = 16
	nestingBudget   = 64
	maxEstimateDeep = 16
	quoteOverhead   = 2
	nullSize        = 4
)

func estimateMapSize(m map[string]any) int64 {
	size := int64(kvOverhead)

	for k, v := range m {
		size += int64(len(k)+kvOverhead) + estimateValueSize(v, 0)
	}

	return size
}

func estimateValueSize(v any, depth int) int64 {
	if depth > maxEstimateDeep {
		return nestingBudget
	}

	switch typed := v.(type) {
	case string:
		return int64(len(typed) + quoteOverhead)
	case json.Number:
		return int64(len(typed))
	case map[string]any:
		size := int64(kvOverhead)
		for k, nested := range typed {
			size += int64(len(k)+kvOverhead) + estimateValueSize(nested, depth+1)
		}

		return size
	case []any:
		size := int64(kvOverhead)
		for _, nested := range typed {
			size += estimateValueSize(nested, depth+1)
		}

		return size
	case nil:
		return nullSize
	default:
		return scalarSize
	}
}

// All implements Reader.
func (s *MemoryStore) All() []CallRecord {
	return s.Filter(FilterOpts{})
}

// Count implements Reader.
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.calls)
}

// Filter implements Reader. Single pass over calls with combined criteria.
func (s *MemoryStore) Filter(opts FilterOpts) []CallRecord {
	return slices.Collect(s.FilterSeq(opts))
}

//nolint:cyclop
func (s *MemoryStore) FilterSeq(opts FilterOpts) iter.Seq[CallRecord] {
	return func(yield func(CallRecord) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		for i := range s.calls {
			c := s.calls[i].call

			if opts.Service != "" && c.Service != opts.Service {
				continue
			}

			if opts.Method != "" && c.Method != opts.Method {
				continue
			}

			if opts.Session != "" && c.Session != "" && c.Session != opts.Session {
				continue
			}

			if opts.ErrorOnly && c.Code == 0 && c.Error == "" {
				continue
			}

			if !yield(c) {
				return
			}
		}
	}
}

// CountFilter counts matching records without materializing them.
func (s *MemoryStore) CountFilter(opts FilterOpts) int {
	total := 0

	for range s.FilterSeq(opts) {
		total++
	}

	return total
}

// FilterWindow returns the newest limit records after skipping offset of them,
// together with the total number of matches. Only the window is held: paginating
// through Filter used to copy the whole history to hand back a page.
func (s *MemoryStore) FilterWindow(opts FilterOpts, limit, offset int) ([]CallRecord, int) {
	offset = max(offset, 0)

	if limit <= 0 {
		matches := s.Filter(opts)

		return matches[:max(len(matches)-offset, 0)], len(matches)
	}

	size := limit + offset
	ring := make([]CallRecord, size)
	total := 0

	for record := range s.FilterSeq(opts) {
		ring[total%size] = record
		total++
	}

	kept := min(total, size)
	ordered := make([]CallRecord, 0, kept)

	for i := range kept {
		ordered = append(ordered, ring[(total-kept+i)%size])
	}

	end := max(len(ordered)-offset, 0)

	return ordered[max(end-limit, 0):end], total
}

// FilterByMethod implements Reader. Delegates to Filter for compatibility.
func (s *MemoryStore) FilterByMethod(service, method string) []CallRecord {
	return s.Filter(FilterOpts{Service: service, Method: method})
}

// AllContext implements context-aware history reader.
func (s *MemoryStore) AllContext(_ context.Context) ([]CallRecord, error) {
	return s.All(), nil
}

// CountContext implements context-aware history reader.
func (s *MemoryStore) CountContext(_ context.Context) (int, error) {
	return s.Count(), nil
}

// FilterByMethodContext implements context-aware history reader.
func (s *MemoryStore) FilterByMethodContext(_ context.Context, service, method string) ([]CallRecord, error) {
	return s.FilterByMethod(service, method), nil
}

// DeleteSession removes records that belong strictly to the provided session.
func (s *MemoryStore) DeleteSession(session string) int {
	if session == "" {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	kept := s.calls[:0]
	deleted := 0

	var bytesAfter int64

	for _, rec := range s.calls {
		if rec.call.Session == session {
			deleted++

			continue
		}

		kept = append(kept, rec)
		bytesAfter += rec.size
	}

	s.calls = kept
	s.currentBytes = bytesAfter

	return deleted
}
