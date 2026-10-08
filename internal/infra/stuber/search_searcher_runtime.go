package stuber

import "github.com/google/uuid"

// upsert inserts or updates stub values by ID.
func (s *searcher) upsert(values ...*Stub) []uuid.UUID {
	for _, value := range values {
		if value != nil {
			normalizeStubHeaders(&value.Headers)
		}
	}

	return s.storage.upsert(values...)
}

// del deletes stubs by ID, returning the number removed.
func (s *searcher) del(ids ...uuid.UUID) int {
	if len(ids) == 0 {
		return 0
	}

	idSet := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for key := range s.stubCallCount {
		if _, ok := idSet[key.id]; ok {
			delete(s.stubCallCount, key)
		}
	}

	return s.storage.del(ids...)
}

// delBySession deletes all stubs belonging to the given session.
func (s *searcher) delBySession(session string) int {
	if session == "" {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for key := range s.stubCallCount {
		if key.session == session {
			delete(s.stubCallCount, key)
		}
	}

	s.lookupMu.Lock()
	delete(s.lookupCache, session)
	s.lookupMu.Unlock()

	return s.storage.delBySession(session)
}

// findByID returns the stub with the given ID, or nil.
func (s *searcher) findByID(id uuid.UUID) *Stub {
	return s.storage.findByID(id)
}

// findBy returns all stubs matching the service and method.
func (s *searcher) findBy(service, method string) ([]*Stub, error) {
	seq, err := s.storage.findAll(service, method)
	if err != nil {
		return nil, s.wrap(err)
	}

	return collectStubs(seq), nil
}

// clear resets call counts, lookup cache and storage.
func (s *searcher) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stubCallCount = make(map[callCountKey]int)

	s.lookupMu.Lock()
	s.lookupCache = make(map[string]*searcherLookup)
	s.lookupMu.Unlock()

	s.storage.clear()
}

// all returns every stored stub.
func (s *searcher) all() []*Stub {
	return collectStubs(s.storage.values())
}

func (s *searcher) sessions() []string {
	return s.storage.sessionsList()
}

// used returns all stubs that have matched at least once.
func (s *searcher) used() []*Stub {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usedIDs := s.collectUsedIDs()

	seq := func(yield func(uuid.UUID) bool) {
		for id := range usedIDs {
			if !yield(id) {
				return
			}
		}
	}

	return collectStubs(s.storage.findByIDs(seq))
}

// unused returns all stubs never matched in any session.
func (s *searcher) unused() []*Stub {
	s.mu.RLock()
	usedIDs := s.collectUsedIDs()
	s.mu.RUnlock()

	var unused []*Stub

	for stub := range s.storage.values() {
		if _, exists := usedIDs[stub.ID]; !exists {
			unused = append(unused, stub)
		}
	}

	return unused
}

// find resolves a Query to a Result (by ID, or optimized service/method search).
func (s *searcher) find(query Query) (*Result, error) {
	query.Headers = normalizeHeaderKeys(query.Headers)

	if query.ID != nil {
		return s.searchByID(query)
	}

	return s.searchOptimized(query)
}

// searchByID retrieves the Stub value associated with the given ID from the searcher.
func (s *searcher) searchByID(query Query) (*Result, error) {
	err := s.ensureServiceMethodExists(query.Service, query.Method)
	if err != nil {
		return nil, err
	}

	_, found := s.lookupVisibleByID(query.Session, *query.ID)
	if found != nil {
		if matchNumber, ok := s.tryReserve(query, found); ok {
			return &Result{found: found, matchNumber: matchNumber}, nil
		}
	}

	return nil, ErrServiceNotFound
}

// tryReserve atomically checks if the stub can be used (under Times limit) and increments the count.
// When query.Session is set, the count is per-session (parallel test isolation).
func (s *searcher) tryReserve(query Query, stub *Stub) (int, bool) {
	if query.RequestInternal() {
		return 1, true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := callCountKey{id: stub.ID, session: query.Session}

	times := stub.EffectiveTimes()
	if times > 0 && s.stubCallCount[key] >= times {
		return 0, false
	}

	s.stubCallCount[key]++

	return s.stubCallCount[key], true
}
