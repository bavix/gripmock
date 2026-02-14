package history

import (
	"encoding/json"
	"iter"
	"slices"
	"strings"
	"sync"
	"time"
)

// CallRecord represents a single gRPC call made to the mock.
type CallRecord struct {
	Service   string         `json:"service,omitempty"`
	Method    string         `json:"method,omitempty"`
	Session   string         `json:"session,omitempty"` // Session ID (empty = global).
	Request   map[string]any `json:"request,omitempty"`
	Response  map[string]any `json:"response,omitempty"`
	Error     string         `json:"error,omitempty"`
	StubID    string         `json:"stubId,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
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
}

// Reader provides read access to recorded calls.
type Reader interface {
	All() []CallRecord
	Count() int
	Filter(opts FilterOpts) []CallRecord
	FilterByMethod(service, method string) []CallRecord
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
	if len(s.redactKeys) > 0 {
		call = redactRecord(call, s.redactKeys)
	}

	if s.messageMaxBytes > 0 {
		call = truncateRecord(call, s.messageMaxBytes)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sz := estimateRecordSize(call)
	s.calls = append(s.calls, call)
	s.currentBytes += sz

	for s.limitBytes > 0 && s.currentBytes > s.limitBytes && len(s.calls) > 0 {
		evicted := s.calls[0]
		s.calls = s.calls[1:]
		s.currentBytes -= estimateRecordSize(evicted)
	}
}

const fallbackRecordSize = 1024

//nolint:gochecknoglobals // immutable, reused for truncation marker
var truncatedMarker = map[string]any{"_truncated": true}

const redactedValue = "[REDACTED]"

func redactRecord(c CallRecord, keys map[string]struct{}) CallRecord {
	if c.Request != nil {
		c.Request = redactMap(c.Request, keys)
	}

	if c.Response != nil {
		c.Response = redactMap(c.Response, keys)
	}

	return c
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
		return arr
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
	if c.Request != nil {
		if b, err := json.Marshal(c.Request); err == nil && int64(len(b)) > maxBytes {
			c.Request = truncatedMarker
		}
	}

	if c.Response != nil {
		if b, err := json.Marshal(c.Response); err == nil && int64(len(b)) > maxBytes {
			c.Response = truncatedMarker
		}
	}

	return c
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

// FilterBySession returns records visible for the given session.
// Session empty: all records (backward compat).
// Session non-empty: records with Session=="" or Session==session.
//
// Deprecated: use MemoryStore.Filter(FilterOpts{Session: session}) for single-pass filtering.
func FilterBySession(records []CallRecord, session string) []CallRecord {
	if session == "" {
		return records
	}

	out := make([]CallRecord, 0, len(records))
	for _, c := range records {
		if c.Session == "" || c.Session == session {
			out = append(out, c)
		}
	}

	return out
}
