package stuber

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
		// Empty query - check if stub can handle empty input
		return len(stub.Input.Equals) == 0 && len(stub.Input.Contains) == 0 && len(stub.Input.Matches) == 0
	}

	if len(query.Input) == 1 {
		return s.fastMatchInput(query.Input[0], stub.Input)
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

	return headersRank
}

// fastMatchInput is an ultra-optimized version of matchInput.
//
//nolint:cyclop
func (s *searcher) fastMatchInput(queryData map[string]any, stubInput InputData) bool {
	// Fast path: empty query
	if len(queryData) == 0 {
		return len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0
	}

	// Ultra-fast path: equals only (most common case)
	if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
		return equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder)
	}

	// Fast path: contains only
	if len(stubInput.Contains) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Matches) == 0 {
		return contains(stubInput.Contains, queryData, stubInput.IgnoreArrayOrder)
	}

	// Fast path: matches only
	if len(stubInput.Matches) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 {
		return matches(stubInput.Matches, queryData, stubInput.IgnoreArrayOrder)
	}

	// Full matching (rare case)
	return matchInput(queryData, stubInput)
}

// fastMatchStream is an ultra-optimized version of matchStreamElements.
//
//nolint:cyclop
func (s *searcher) fastMatchStream(queryStream []map[string]any, stubStream []InputData) bool {
	// Check if stub has any input matching conditions
	hasConditions := false

	for _, stubElement := range stubStream {
		if stubElement.Equals != nil || stubElement.Contains != nil || stubElement.Matches != nil {
			hasConditions = true

			break
		}
	}

	if !hasConditions {
		return false // Stub has no input matching conditions
	}

	// Fast path: empty query stream
	if len(queryStream) == 0 {
		// Check if all stub stream elements can handle empty input
		for _, stubElement := range stubStream {
			if len(stubElement.Equals) > 0 || len(stubElement.Contains) > 0 || len(stubElement.Matches) > 0 {
				return false
			}
		}

		return true
	}

	// Fast path: single element
	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastMatchInput(queryStream[0], stubStream[0])
	}

	// Use original implementation for complex cases
	return matchStreamElements(queryStream, stubStream)
}

// fastRankInput is an ultra-optimized version of rankInput.
func (s *searcher) fastRankInput(queryData map[string]any, stubInput InputData) float64 {
	// Fast path: empty query
	if len(queryData) == 0 {
		// Check if stub can handle empty input
		if len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
			return 1.0 // Perfect match for empty input
		}

		return 0
	}

	// Fast path: equals only
	if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
		if equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder) {
			return 1.0
		}

		return 0
	}

	// Use original implementation for complex cases
	return rankInput(queryData, stubInput)
}

// fastRankStream is an ultra-optimized version of rankStreamElements.
func (s *searcher) fastRankStream(queryStream []map[string]any, stubStream []InputData) float64 {
	// Fast path: empty query stream
	if len(queryStream) == 0 {
		// Check if all stub stream elements can handle empty input
		for _, stubElement := range stubStream {
			if len(stubElement.Equals) > 0 || len(stubElement.Contains) > 0 || len(stubElement.Matches) > 0 {
				return 0
			}
		}

		return 1.0 // Perfect match for empty input
	}

	// Fast path: single element
	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastRankInput(queryStream[0], stubStream[0])
	}

	// Use original implementation for complex cases
	return rankStreamElements(queryStream, stubStream)
}
