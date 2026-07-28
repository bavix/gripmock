package history

import (
	"context"
	"encoding/json"
	"iter"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CallRecord represents a single gRPC call made to the mock.
type CallRecord struct {
	StubID    uuid.UUID        `json:"stubId,omitempty"`
	Service   string           `json:"service,omitempty"`
	Method    string           `json:"method,omitempty"`
	Session   string           `json:"session,omitempty"`   // Session ID (empty = global).
	Request   map[string]any   `json:"request,omitempty"`   // Deprecated: use Requests.
	Requests  []map[string]any `json:"requests,omitempty"`  // For streaming calls with multiple messages.
	Response  map[string]any   `json:"response,omitempty"`  // Deprecated: use Responses.
	Responses []map[string]any `json:"responses,omitempty"` // For streaming calls with multiple messages.
	Code      uint32           `json:"code,omitempty"`      // gRPC status code (e.g., codes.OK, codes.NotFound).
	Error     string           `json:"error,omitempty"`
	ElapsedMS int64            `json:"elapsedMs,omitempty"` // Handler duration in milliseconds.
	Timestamp time.Time        `json:"timestamp"`
}

// Recorder records gRPC calls for inspection and verification.
type Recorder interface {
	Record(call CallRecord)
}

// FilterOpts specifies filter criteria for recorded calls.
// Empty string means "no filter" for that field.
// Session non-empty: records with Session=="" or Session==Session (visible to session).
type FilterOpts struct {
	Service string
	Method  string
	Session string

	// ErrorOnly keeps only calls that ended with a non-OK gRPC status.
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
// LimitBytes 0 means unlimited. MessageMaxBytes 0 means no truncation.
type MemoryStore struct {
	mu              sync.RWMutex
	calls           []CallRecord
	limitBytes      int64
	messageMaxBytes int64
	redactKeys      map[string]struct{} // lowercased keys to redact
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

// WithRedactKeys replaces values for matching keys (case-insensitive) with "[REDACTED]"
// in Request/Response. Keys are matched at any nesting level.
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
// limitBytes <= 0 means unlimited.
func NewMemoryStore(limitBytes int64, opts ...MemoryStoreOption) *MemoryStore {
	s := &MemoryStore{limitBytes: limitBytes}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Record implements Recorder.
func (s *MemoryStore) Record(call CallRecord) {
	// The store must OWN its message maps: callers may reuse/mutate the maps they
	// passed. Redaction already deep-copies every map, so only the non-redacting
	// path needs an explicit clone (truncation shares non-truncated maps).
	if len(s.redactKeys) > 0 {
		call = redactRecord(call, s.redactKeys)
	} else {
		call = cloneRecordMessages(call)
	}

	if s.messageMaxBytes > 0 {
		call = truncateRecord(call, s.messageMaxBytes)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sz := estimateRecordSize(call)
	s.calls = append(s.calls, call)
	s.currentBytes += sz

	// Keep at least the newest record: if a single call alone exceeds limitBytes,
	// evicting it too would silently drop the most recent (and only) record.
	for s.limitBytes > 0 && s.currentBytes > s.limitBytes && len(s.calls) > 1 {
		evicted := s.calls[0]
		s.calls = s.calls[1:]
		s.currentBytes -= estimateRecordSize(evicted)
	}
}

// Clear removes all recorded calls, resetting the store to empty in place.
// The store pointer stays valid, so a running server holding this recorder
// keeps writing into the same (now-empty) store.
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = nil
	s.currentBytes = 0
}

const fallbackRecordSize = 1024

const redactedValue = "[REDACTED]"

// freshTruncatedMarker returns a new marker map per call. A shared global would
// be aliased into every truncated record, so any consumer mutating it would
// corrupt all of them.
func freshTruncatedMarker() map[string]any {
	return map[string]any{"_truncated": true}
}

// redactRecord strips configured secret keys from ALL message fields. Real
// callers populate the plural Requests/Responses (the primary fields the UI and
// API read); the deprecated singular Request/Response mirror the first message.
// Redacting only the singular fields (the previous behavior) left every stored
// message — including Requests[0], which aliases the same map — unredacted.
func redactRecord(c CallRecord, keys map[string]struct{}) CallRecord {
	c.Requests = redactMaps(c.Requests, keys)
	c.Responses = redactMaps(c.Responses, keys)

	if c.Request != nil {
		c.Request = redactMap(c.Request, keys)
	}

	if c.Response != nil {
		c.Response = redactMap(c.Response, keys)
	}

	// Keep the deprecated singular fields consistent with the redacted slices.
	if len(c.Requests) > 0 {
		c.Request = c.Requests[0]
	}

	if len(c.Responses) > 0 {
		c.Response = c.Responses[0]
	}

	return c
}

// cloneRecordMessages deep-copies the message maps so the stored record does not
// alias caller-owned maps (which the caller may mutate or reuse afterwards).
func cloneRecordMessages(c CallRecord) CallRecord {
	c.Request = cloneMap(c.Request)
	c.Response = cloneMap(c.Response)
	c.Requests = cloneMaps(c.Requests)
	c.Responses = cloneMaps(c.Responses)

	return c
}

// mapEach applies fn to each message map, returning a new slice. A nil/empty
// input is returned unchanged. Shared by the clone/redact/truncate passes.
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

// truncateRecord replaces any message exceeding maxBytes with a marker. It must
// cover the plural Requests/Responses (the primary fields); truncating only the
// deprecated singular fields left the size guard dead for the real data path.
func truncateRecord(c CallRecord, maxBytes int64) CallRecord {
	c.Requests = truncateMaps(c.Requests, maxBytes)
	c.Responses = truncateMaps(c.Responses, maxBytes)

	if c.Request != nil {
		c.Request = truncateMessage(c.Request, maxBytes)
	}

	if c.Response != nil {
		c.Response = truncateMessage(c.Response, maxBytes)
	}

	// Keep the deprecated singular fields consistent with the truncated slices.
	if len(c.Requests) > 0 {
		c.Request = c.Requests[0]
	}

	if len(c.Responses) > 0 {
		c.Response = c.Responses[0]
	}

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

func estimateRecordSize(c CallRecord) int64 {
	b, err := json.Marshal(c)
	if err != nil {
		return fallbackRecordSize
	}

	return int64(len(b))
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

// FilterSeq returns an iterator over records matching FilterOpts.
// Single pass, no intermediate allocations. Lock held during iteration.
//
//nolint:cyclop
func (s *MemoryStore) FilterSeq(opts FilterOpts) iter.Seq[CallRecord] {
	return func(yield func(CallRecord) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		for _, c := range s.calls {
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
// Global records (Session == "") are not affected.
func (s *MemoryStore) DeleteSession(session string) int {
	if session == "" {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	kept := s.calls[:0]
	deleted := 0

	var bytesAfter int64

	for _, c := range s.calls {
		if c.Session == session {
			deleted++

			continue
		}

		kept = append(kept, c)
		bytesAfter += estimateRecordSize(c)
	}

	s.calls = kept
	s.currentBytes = bytesAfter

	return deleted
}
