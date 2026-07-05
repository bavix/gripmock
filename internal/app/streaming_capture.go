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
	responses        []map[string]any
	lastResponseTime time.Time
	startTime        time.Time
	recordDelay      bool
}

func NewStreamCaptureState() *StreamCaptureState {
	return &StreamCaptureState{
		requests:  make([]map[string]any, 0, proxyMessagesInitCap),
		responses: make([]map[string]any, 0, proxyMessagesInitCap),
	}
}

func (s *StreamCaptureState) AppendRequest(req map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = append(s.requests, req)
}

func (s *StreamCaptureState) AppendResponseWithTiming(resp map[string]any, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recordDelay && !s.lastResponseTime.IsZero() {
		delay := now.Sub(s.lastResponseTime)
		resp[stuber.GripMockKey] = map[string]any{
			"delay": delay.String(),
		}
	} else if s.recordDelay && s.lastResponseTime.IsZero() {
		delay := now.Sub(s.startTime)
		resp[stuber.GripMockKey] = map[string]any{
			"delay": delay.String(),
		}
	}

	s.responses = append(s.responses, resp)
	s.lastResponseTime = now
}

func (s *StreamCaptureState) Snapshot() ([]map[string]any, []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.requests),
		slices.Clone(s.responses)
}

// HasTimedResponses returns true if at least one response was captured with per-element delay.
func (s *StreamCaptureState) HasTimedResponses() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.recordDelay && len(s.responses) > 0
}
