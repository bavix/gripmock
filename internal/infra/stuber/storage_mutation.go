package stuber

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

// upsert inserts or updates the given Stubs in storage.
// Optimized for minimal allocations and maximum performance.
func (s *storage) upsert(values ...*Stub) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}

	// Pre-allocate with exact size to minimize allocations
	results := make([]uuid.UUID, len(values))

	s.mu.Lock()
	defer s.mu.Unlock()

	// Process all values in a single pass (direct field access for performance)
	for i, v := range values {
		results[i] = v.ID

		if old, exists := s.itemsByID[v.ID]; exists {
			s.removeStubIndexes(old)
		}

		leftID := s.id(v.Service)
		rightID := s.id(v.Method)
		index := s.pos(leftID, rightID)

		if s.items[index] == nil {
			s.items[index] = make(map[uuid.UUID]*Stub, 1)
		}

		s.items[index][v.ID] = v
		s.upsertSessionIndex(s.itemSorted, index, v.Session, v)
		s.upsertMethodSessionIndex(rightID, v.Session, v)
		s.incrementSession(v.Session)
		s.itemsByID[v.ID] = v
		s.lefts[leftID] = struct{}{}
	}

	return results
}

// del deletes the Stub values with the given UUIDs from the storage.
// It returns the number of Stub values that were successfully deleted.
func (s *storage) del(keys ...uuid.UUID) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := 0

	for _, key := range keys {
		if v, ok := s.itemsByID[key]; ok {
			s.removeStubIndexes(v)
			delete(s.itemsByID, key)

			deleted++
		}
	}

	return deleted
}

// delBySession deletes all Stub values belonging to the given session.
// It returns the number of Stub values that were deleted.
func (s *storage) delBySession(session string) int {
	if session == "" {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int

	for id, stub := range s.itemsByID {
		if stub.Session == session {
			s.removeStubIndexes(stub)
			delete(s.itemsByID, id)

			deleted++
		}
	}

	return deleted
}

func (s *storage) removeStubIndexes(stub *Stub) {
	pos := s.pos(s.id(stub.Service), s.id(stub.Method))

	if m, exists := s.items[pos]; exists {
		delete(m, stub.ID)

		if len(m) == 0 {
			delete(s.items, pos)
		}
	}

	s.removeSessionIndex(s.itemSorted, pos, stub.Session, stub.ID)
	methodID := s.id(stub.Method)
	s.removeMethodSessionIndex(methodID, stub.Session, stub.ID)
	s.decrementSession(stub.Session)
}

func (s *storage) incrementSession(session string) {
	if session == "" {
		return
	}

	s.sessions[session]++
}

func (s *storage) decrementSession(session string) {
	if session == "" {
		return
	}

	next := s.sessions[session] - 1
	if next <= 0 {
		delete(s.sessions, session)

		return
	}

	s.sessions[session] = next
}

func (s *storage) sessionsList() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := slices.Collect(maps.Keys(s.sessions))
	if sessions == nil {
		return []string{}
	}

	slices.Sort(sessions)

	return sessions
}

func (s *storage) upsertSessionIndex(
	sorted map[uint64]map[string][]*Stub,
	key uint64,
	session string,
	stub *Stub,
) {
	sortedBuckets := sorted[key]
	if sortedBuckets == nil {
		sortedBuckets = make(map[string][]*Stub)
		sorted[key] = sortedBuckets
	}

	sortedBuckets[session] = append(sortedBuckets[session], stub)
}

func (s *storage) upsertMethodSessionIndex(key uint32, session string, stub *Stub) {
	sortedBuckets := s.methodSorted[key]
	if sortedBuckets == nil {
		sortedBuckets = make(map[string][]*Stub)
		s.methodSorted[key] = sortedBuckets
	}

	sortedBuckets[session] = append(sortedBuckets[session], stub)
}

func (s *storage) removeSessionIndex(
	sorted map[uint64]map[string][]*Stub,
	key uint64,
	session string,
	id uuid.UUID,
) {
	sortedBuckets, exists := sorted[key]
	if !exists {
		return
	}

	sortedBuckets[session] = removeSortedStubByID(sortedBuckets[session], id)
	if len(sortedBuckets[session]) == 0 {
		delete(sortedBuckets, session)
	}

	if len(sortedBuckets) == 0 {
		delete(sorted, key)
	}
}

func (s *storage) removeMethodSessionIndex(key uint32, session string, id uuid.UUID) {
	sortedBuckets, exists := s.methodSorted[key]
	if !exists {
		return
	}

	sortedBuckets[session] = removeSortedStubByID(sortedBuckets[session], id)
	if len(sortedBuckets[session]) == 0 {
		delete(sortedBuckets, session)
	}

	if len(sortedBuckets) == 0 {
		delete(s.methodSorted, key)
	}
}
