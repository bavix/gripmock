package stuber

// The equals index turns the common "one stub per input value" setup from a
// full scan of every stub registered for a service/method into a map lookup.
//
// It is a pure fast path: it only ever narrows the candidate set, never
// changes how candidates are matched or ranked, and the caller falls back to
// the full set whenever the index cannot answer (see searcher.searchOptimized).
//
// Soundness. A stub is indexed only when its input is a plain `equals` block
// with string values and no other matcher kind. Such a stub matches a query
// only if *every* one of its equals pairs is present in the query, so in
// particular its indexed pair must be. Probing the query's own fields
// therefore cannot miss a stub that would have matched. String expectations
// compare by exact equality (see fieldValueEquals), so a string-keyed map is
// an exact lookup rather than an approximation.

// indexKeyFor reports the field a stub is indexed under -- its
// lexicographically smallest equals key -- and whether it can be indexed at
// all. Using a single, deterministic key keeps each stub in exactly one
// bucket, so a lookup never returns the same stub twice.
func indexKeyFor(stub *Stub) (string, string, bool) {
	if !isIndexableInput(stub) {
		return "", "", false
	}

	var key, value string

	first := true

	for k, v := range stub.Input.Equals {
		s, isString := v.(string)
		if !isString {
			return "", "", false
		}

		if first || k < key {
			key, value, first = k, s, false
		}
	}

	return key, value, true
}

// isIndexableInput reports whether a stub's input is a plain equals block --
// the only shape the index can reason about exactly.
func isIndexableInput(stub *Stub) bool {
	if stub.Inputs != nil {
		return false
	}

	in := stub.Input

	return len(in.Equals) > 0 &&
		len(in.Contains) == 0 &&
		len(in.Matches) == 0 &&
		len(in.Glob) == 0 &&
		len(in.AnyOf) == 0
}

// indexStub adds a stub to the per-(service, method) equals index, or to the
// unindexed list when it cannot be indexed.
func (s *storage) indexStub(index uint64, stub *Stub) {
	key, value, ok := indexKeyFor(stub)
	if !ok {
		s.unindexed[index] = append(s.unindexed[index], stub)

		return
	}

	byField := s.equalsIndex[index]
	if byField == nil {
		byField = make(map[string]map[string][]*Stub, 1)
		s.equalsIndex[index] = byField
	}

	byValue := byField[key]
	if byValue == nil {
		byValue = make(map[string][]*Stub, 1)
		byField[key] = byValue
	}

	byValue[value] = append(byValue[value], stub)
}

// deindexStub reverses indexStub.
func (s *storage) deindexStub(index uint64, stub *Stub) {
	key, value, ok := indexKeyFor(stub)
	if !ok {
		s.unindexed[index] = removeSortedStubByID(s.unindexed[index], stub.ID)
		if len(s.unindexed[index]) == 0 {
			delete(s.unindexed, index)
		}

		return
	}

	byField, exists := s.equalsIndex[index]
	if !exists {
		return
	}

	byValue, exists := byField[key]
	if !exists {
		return
	}

	byValue[value] = removeSortedStubByID(byValue[value], stub.ID)
	if len(byValue[value]) == 0 {
		delete(byValue, value)
	}

	if len(byValue) == 0 {
		delete(byField, key)
	}

	if len(byField) == 0 {
		delete(s.equalsIndex, index)
	}
}

// indexedCandidates returns the stubs that could match queryData, or ok=false
// when the index cannot be used for this query (multi-message streams, empty
// input, or a service/method that has no indexed stubs at all). The result
// still has to go through the normal matcher: it is a superset filter, not a
// match.
func (s *storage) indexedCandidates(indexes []uint64, queryData map[string]any) ([]*Stub, bool) {
	if len(queryData) == 0 {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var (
		candidates []*Stub
		usable     bool
	)

	for _, index := range indexes {
		byField, exists := s.equalsIndex[index]
		if !exists {
			continue
		}

		usable = true

		candidates = append(candidates, s.unindexed[index]...)

		for queryKey, queryValue := range queryData {
			value, isString := queryValue.(string)
			if !isString {
				continue
			}

			// equals() resolves a stub key against the query through
			// findValueWithVariations, so a stub key K matches query key Q
			// when Q is K, camelCase(K) or snake_case(K). Probing those three
			// forms of Q covers every K that could have matched.
			for _, key := range keyVariations(queryKey) {
				candidates = append(candidates, byField[key][value]...)
			}
		}
	}

	if !usable {
		return nil, false
	}

	return candidates, true
}

// keyVariations returns the stub-side key spellings that a query key can
// resolve to, without duplicates.
func keyVariations(queryKey string) []string {
	variations := [3]string{queryKey, toSnakeCase(queryKey), toCamelCase(queryKey)}
	unique := variations[:1]

	for _, candidate := range variations[1:] {
		if candidate != "" && candidate != variations[0] && (len(unique) < 2 || candidate != unique[1]) {
			unique = append(unique, candidate)
		}
	}

	return unique
}
