package stuber

import "slices"

// inputHasConditions reports whether an input element carries a real matcher
// requirement. AnyOf counts only when at least one alternative actually declares
// a field matcher — an anyOf of empty alternatives imposes no requirement.
func inputHasConditions(e InputData) bool {
	return len(e.Equals) > 0 ||
		len(e.Contains) > 0 ||
		len(e.Matches) > 0 ||
		len(e.Glob) > 0 ||
		slices.ContainsFunc(e.AnyOf, anyOfElementHasFields)
}

// anyOfElementHasFields reports whether an alternative sets any matcher entries
// (len-based: an empty {} matcher is not a requirement).
func anyOfElementHasFields(alt AnyOfElement) bool {
	return len(alt.Equals) > 0 || len(alt.Contains) > 0 || len(alt.Matches) > 0 || len(alt.Glob) > 0
}

// anyOfElementDeclares reports whether an alternative declares any matcher map
// (nil-based: an empty {} still counts as a declaration).
func anyOfElementDeclares(alt AnyOfElement) bool {
	return alt.Equals != nil || alt.Contains != nil || alt.Matches != nil || alt.Glob != nil
}

// fastMatchV2 is an ultra-optimized version of matchV2.
//
//nolint:cyclop
func (s *searcher) fastMatchV2(query Query, stub *Stub) bool {
	// If stub has headers, query must also have headers
	if stub.Headers.Len() > 0 && len(query.Headers) == 0 {
		return false
	}

	if len(query.Headers) > 0 && !matchHeaders(query.Headers, stub.Headers) {
		return false
	}

	// Priority to Inputs (stream) over Input (unary)
	// stub.Inputs != nil means stream stub (even if empty slice)
	if stub.Inputs != nil {
		if len(stub.Inputs) == 0 {
			return false // stream stub with no patterns matches nothing
		}

		return s.fastMatchStream(query.Input, stub.Inputs)
	}

	// Handle Input (unary) - stub uses Input
	// Stub with no input conditions matches any query (including empty)
	if len(query.Input) == 0 {
		// Empty query - a stub matches only if it has no input conditions at all
		// (glob/anyOf included, else a glob-only stub falsely matches empty input).
		return !inputHasConditions(stub.Input)
	}

	if len(query.Input) == 1 {
		return s.fastMatchInput(query.Input[0], stub.Input)
	}

	// Multi-message stream against a single-Input stub (legacy client-stream style).
	// Try each message newest-first; return true on first match.
	for _, v := range slices.Backward(query.Input) {
		if s.fastMatchInput(v, stub.Input) {
			return true
		}
	}

	return false
}

// fastRankV2 is an ultra-optimized version of rankMatchV2.
func (s *searcher) fastRankV2(query Query, stub *Stub) float64 {
	if len(query.Headers) > 0 && !matchHeaders(query.Headers, stub.Headers) {
		return 0
	}

	// Include header rank so that stubs with matching headers get higher score within same priority
	headersRank := rankHeaders(query.Headers, stub.Headers)

	// Priority to Inputs (stream) over Input (unary)
	if stub.Inputs != nil {
		if len(stub.Inputs) == 0 {
			return headersRank
		}

		inputsBonus := 1000.0

		return headersRank + s.fastRankStream(query.Input, stub.Inputs) + inputsBonus
	}

	// Handle Input (unary)
	if len(query.Input) == 0 {
		// Empty query - return header rank only
		return headersRank
	}

	if len(query.Input) == 1 {
		return headersRank + s.fastRankInput(query.Input[0], stub.Input)
	}

	// Multi-message stream against a single-Input stub: best score across messages.
	// Weight by position so that a match on a later message scores higher than an
	// earlier one, making the stub that matches the last message win.
	n := len(query.Input)
	best := 0.0

	for i := n - 1; i >= 0; i-- {
		r := s.fastRankInput(query.Input[i], stub.Input)
		if r > 0 {
			weighted := r * (float64(i+1) / float64(n))
			if weighted > best {
				best = weighted
			}
		}
	}

	return headersRank + best
}

// fastMatchInput is an ultra-optimized version of matchInput.
//
//nolint:cyclop
func (s *searcher) fastMatchInput(queryData map[string]any, stubInput InputData) bool {
	// Fast path: empty query
	if len(queryData) == 0 {
		return !inputHasConditions(stubInput)
	}

	// Skip single-condition fast paths when AnyOf or Glob is present — either would
	// be silently dropped here, so fall through to the full matchInput which ANDs
	// every matcher kind (glob included).
	if len(stubInput.AnyOf) == 0 && len(stubInput.Glob) == 0 {
		// Ultra-fast path: equals only (most common case)
		if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
			return equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder)
		}

		// Fast path: contains only
		if len(stubInput.Contains) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Matches) == 0 {
			return contains(stubInput.Contains, queryData)
		}

		// Fast path: matches only
		if len(stubInput.Matches) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 {
			return matches(stubInput.Matches, queryData)
		}
	}

	// Full matching (handles AnyOf and combined conditions)
	return matchInput(queryData, stubInput)
}

// fastMatchStream is an ultra-optimized version of matchStreamElements.
//
//nolint:cyclop
func (s *searcher) fastMatchStream(queryStream []map[string]any, stubStream []InputData) bool {
	// A stub element "declares" a matcher when the map is non-nil (even if empty:
	// an empty {} means "explicitly matches empty input"). Glob and AnyOf count
	// too — for AnyOf only when some alternative actually declares a matcher, so a
	// glob- or anyOf-only stream stub is recognized while an empty anyOf is not.
	declaresMatcher := func(e InputData) bool {
		return e.Equals != nil || e.Contains != nil || e.Matches != nil || e.Glob != nil ||
			slices.ContainsFunc(e.AnyOf, anyOfElementDeclares)
	}

	if !slices.ContainsFunc(stubStream, declaresMatcher) {
		return false // Stub has no input matching conditions
	}

	// Fast path: empty query stream. A stub matches only if no element REQUIRES
	// fields (an empty {} matcher still matches empty input). Reuse
	// inputHasConditions so the len-based kind list stays a single source of truth.
	if len(queryStream) == 0 {
		return !slices.ContainsFunc(stubStream, inputHasConditions)
	}

	// Fast path: single element
	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastMatchInput(queryStream[0], stubStream[0])
	}

	// Broadcast fast path: single pattern, multiple messages — each must match.
	if len(stubStream) == 1 && len(queryStream) > 1 {
		for i := range queryStream {
			if !s.fastMatchInput(queryStream[i], stubStream[0]) {
				return false
			}
		}

		return true
	}

	// Use original implementation for complex cases
	return matchStreamElements(queryStream, stubStream)
}

// fastRankInput is an ultra-optimized version of rankInput.
func (s *searcher) fastRankInput(queryData map[string]any, stubInput InputData) float64 {
	// Fast path: empty query
	if len(queryData) == 0 {
		// Check if stub can handle empty input (glob/anyOf included, else a
		// non-matching glob/anyOf-only stub scores a spurious perfect 1.0).
		if !inputHasConditions(stubInput) {
			return 1.0 // Perfect match for empty input
		}

		return 0
	}

	// Fast path: equals only (Glob/AnyOf absent — otherwise glob/anyOf rank is dropped).
	if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 &&
		len(stubInput.Glob) == 0 && len(stubInput.AnyOf) == 0 {
		if equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder) {
			return 1.0
		}

		return 0
	}

	// Use original implementation for complex cases
	return rankInput(queryData, stubInput)
}

// fastRankStreamBroadcast ranks a multi-message stream against a single stub pattern.
func (s *searcher) fastRankStreamBroadcast(queryStream []map[string]any, pattern InputData) float64 {
	n := len(queryStream)

	var total float64

	for _, msg := range queryStream {
		total += s.fastRankInput(msg, pattern)
	}

	return total / float64(n)
}

// fastRankStream is an ultra-optimized version of rankStreamElements.
func (s *searcher) fastRankStream(queryStream []map[string]any, stubStream []InputData) float64 {
	// Fast path: empty query stream — a stub ranks 1.0 only if no element requires
	// fields. Mirror fastMatchStream (glob/anyOf included) so match and rank agree;
	// otherwise a glob-/anyOf-only stub that does NOT match still scores a perfect 1.0.
	if len(queryStream) == 0 {
		if slices.ContainsFunc(stubStream, inputHasConditions) {
			return 0
		}

		return 1.0
	}

	// Fast path: single element
	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastRankInput(queryStream[0], stubStream[0])
	}

	// Broadcast fast path: single pattern, multiple messages — average rank.
	if len(stubStream) == 1 && len(queryStream) > 1 {
		return s.fastRankStreamBroadcast(queryStream, stubStream[0])
	}

	// Use original implementation for complex cases
	return rankStreamElements(queryStream, stubStream)
}
