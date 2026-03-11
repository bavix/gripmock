package stuber

import (
	"iter"

	"github.com/google/uuid"
)

func (s *searcher) collectUsedIDs() map[uuid.UUID]struct{} {
	usedIDs := make(map[uuid.UUID]struct{}, len(s.stubCallCount))

	for key, n := range s.stubCallCount {
		if n > 0 {
			usedIDs[key.id] = struct{}{}
		}
	}

	return usedIDs
}

func (s *searcher) isVisibleAndNotExhausted(stub *Stub, session string) bool {
	if !isStubVisibleForSession(stub.Session, session) {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.notExhausted(stub, session)
}

// filterExhaustedStubs removes stubs that have reached their Times limit for the given session.
func (s *searcher) filterExhaustedStubs(stubs []*Stub, session string) []*Stub {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := stubs[:0]
	for _, stub := range stubs {
		if s.notExhausted(stub, session) {
			filtered = append(filtered, stub)
		}
	}

	return filtered
}

func (s *searcher) filterNotExhaustedSeq(seq iter.Seq[*Stub], session string) iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		for stub := range seq {
			if s.notExhausted(stub, session) {
				if !yield(stub) {
					return
				}
			}
		}
	}
}

func (s *searcher) notExhausted(stub *Stub, session string) bool {
	times := stub.EffectiveTimes()
	if times <= 0 {
		return true
	}

	key := callCountKey{id: stub.ID, session: session}

	return s.stubCallCount[key] < times
}

// filterBySession returns stubs visible for the given session.
// Session empty: only global stubs (stub.Session == "").
// Session non-empty: global stubs + stubs for that session.
func filterBySession(stubs []*Stub, session string) []*Stub {
	filtered := stubs[:0]
	for _, stub := range stubs {
		if isStubVisibleForSession(stub.Session, session) {
			filtered = append(filtered, stub)
		}
	}

	return filtered
}
