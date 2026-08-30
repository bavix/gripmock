package stuber

import "regexp/syntax"

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
	fields, pattern, ok := indexableFields(stub)
	if !ok {
		return "", "", false
	}

	var key, value string

	first := true

	for k, v := range fields {
		s, isString := v.(string)
		if !isString {
			return "", "", false
		}

		if pattern {
			literal, isLiteral := anchoredLiteral(s)
			if !isLiteral {
				return "", "", false
			}

			s = literal
		}

		if first || k < key {
			key, value, first = k, s, false
		}
	}

	return key, value, true
}

func anchoredLiteral(pattern string) (string, bool) {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", false
	}

	parsed = parsed.Simplify()
	if parsed.Op != syntax.OpConcat || len(parsed.Sub) < 3 {
		return "", false
	}

	subs := parsed.Sub
	if subs[0].Op != syntax.OpBeginText || subs[len(subs)-1].Op != syntax.OpEndText {
		return "", false
	}

	var literal []rune

	for _, sub := range subs[1 : len(subs)-1] {
		if sub.Op != syntax.OpLiteral || sub.Flags&syntax.FoldCase != 0 {
			return "", false
		}

		literal = append(literal, sub.Rune...)
	}

	if len(literal) == 0 {
		return "", false
	}

	return string(literal), true
}

// isIndexableInput reports whether a stub's input is a plain equals block --
// the only shape the index can reason about exactly.
func indexableFields(stub *Stub) (map[string]any, bool, bool) {
	if stub.DeclaresStreamMatchers() {
		return nil, false, false
	}

	in := stub.Matchers()[0]
	if len(in.Glob) > 0 || len(in.AnyOf) > 0 {
		return nil, false, false
	}

	switch {
	case len(in.Matches) > 0:
		if len(in.Equals) > 0 || len(in.Contains) > 0 {
			return nil, false, false
		}

		return in.Matches, true, true
	case len(in.Equals) > 0:
		return in.Equals, false, true
	case len(in.Contains) > 0:
		return in.Contains, false, true
	}

	return nil, false, false
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
		spellings  [3]string
	)

	for _, index := range indexes {
		candidates = append(candidates, s.unindexed[index]...)

		byField, exists := s.equalsIndex[index]
		if !exists {
			continue
		}

		usable = true

		for queryKey, queryValue := range queryData {
			value, isString := queryValue.(string)
			if !isString {
				continue
			}

			// equals() resolves a stub key against the query through
			// findValueWithVariations, so a stub key K matches query key Q
			// when Q is K, camelCase(K) or snake_case(K). Probing those three
			// forms of Q covers every K that could have matched.
			for _, key := range keyVariations(queryKey, &spellings) {
				candidates = append(candidates, byField[key][value]...)
			}
		}
	}

	if !usable {
		return nil, false
	}

	return candidates, true
}

// keyVariations fills dst with the distinct stub-side spellings a query key can
// resolve to and returns the filled prefix. The caller owns dst so probing a
// query costs no allocation.
func keyVariations(queryKey string, dst *[3]string) []string {
	dst[0] = queryKey
	filled := 1

	for _, candidate := range [2]string{toSnakeCase(queryKey), toCamelCase(queryKey)} {
		if candidate == "" {
			continue
		}

		duplicate := false

		for i := range filled {
			if dst[i] == candidate {
				duplicate = true

				break
			}
		}

		if !duplicate {
			dst[filled] = candidate
			filled++
		}
	}

	return dst[:filled]
}
