package app

import (
	"sync"
)

type StreamCaptureState struct {
	mu        sync.Mutex
	requests  []map[string]any
	responses []map[string]any
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

func (s *StreamCaptureState) AppendResponse(resp map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.responses = append(s.responses, resp)
}

func (s *StreamCaptureState) Snapshot() ([]map[string]any, []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]map[string]any(nil), s.requests...),
		append([]map[string]any(nil), s.responses...)
}
