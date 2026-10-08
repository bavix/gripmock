package app

import (
	"slices"
	"sync"
	"time"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type StreamCaptureState struct {
	mu               sync.Mutex
	requests         []map[string]any
	responses        []any
	lastResponseTime time.Time
	startTime        time.Time
	recordDelay      bool
	limit            int
}

func NewStreamCaptureState() *StreamCaptureState {
	return &StreamCaptureState{
		requests:  make([]map[string]any, 0, proxyMessagesInitCap),
		responses: make([]any, 0, proxyMessagesInitCap),
	}
}

func (s *StreamCaptureState) SetLimit(limit int) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.limit = limit
}

func (s *StreamCaptureState) AppendRequest(req map[string]any) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.full(len(s.requests)) {
		return
	}

	s.requests = append(s.requests, req)
}

func (s *StreamCaptureState) AppendResponseWithTiming(resp any, now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.full(len(s.responses)) {
		s.lastResponseTime = now

		return
	}

	if respMap, ok := resp.(map[string]any); ok && s.recordDelay {
		since := s.lastResponseTime
		if since.IsZero() {
			since = s.startTime
		}

		respMap[stuber.GripMockKey] = map[string]any{
			"delay": now.Sub(since).String(),
		}
	}

	s.responses = append(s.responses, resp)
	s.lastResponseTime = now
}

func (s *StreamCaptureState) Snapshot() ([]map[string]any, []any) {
	if s == nil {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.requests),
		slices.Clone(s.responses)
}

// HasTimedResponses returns true if at least one response was captured with per-element delay.
func (s *StreamCaptureState) HasTimedResponses() bool {
	if s == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.recordDelay && len(s.responses) > 0
}

func (s *StreamCaptureState) full(count int) bool {
	return s.limit > 0 && count >= s.limit
}
