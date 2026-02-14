package history

import (
	"sync"
	"time"
)

// CallRecord represents a single gRPC call made to the mock.
type CallRecord struct {
	Service   string
	Method    string
	Session   string // Session ID (empty = global).
	Request   map[string]any
	Response  map[string]any
	Error     string
	StubID    string
	Timestamp time.Time
}

// Recorder records gRPC calls for inspection and verification.
type Recorder interface {
	Record(call CallRecord)
}

// Reader provides read access to recorded calls.
type Reader interface {
	All() []CallRecord
	Count() int
	FilterByMethod(service, method string) []CallRecord
}

// MemoryStore implements both Recorder and Reader (in-memory).
type MemoryStore struct {
	mu    sync.RWMutex
	calls []CallRecord
}

// Record implements Recorder.
func (s *MemoryStore) Record(call CallRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, call)
}

// All implements Reader.
func (s *MemoryStore) All() []CallRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]CallRecord, len(s.calls))
	copy(out, s.calls)

	return out
}

// Count implements Reader.
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.calls)
}

// FilterByMethod implements Reader.
func (s *MemoryStore) FilterByMethod(service, method string) []CallRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []CallRecord

	for _, c := range s.calls {
		if c.Service == service && c.Method == method {
			out = append(out, c)
		}
	}

	return out
}

// FilterBySession returns records visible for the given session.
// Session empty: all records (backward compat).
// Session non-empty: records with Session=="" or Session==session.
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
