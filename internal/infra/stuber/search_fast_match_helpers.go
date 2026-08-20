package stuber

import "slices"

func inputHasConditions(e InputData) bool {
	return len(e.Equals) > 0 ||
		len(e.Contains) > 0 ||
		len(e.Matches) > 0 ||
		len(e.Glob) > 0 ||
		slices.ContainsFunc(e.AnyOf, anyOfElementHasFields)
}

func anyOfElementHasFields(alt AnyOfElement) bool {
	return len(alt.Equals) > 0 || len(alt.Contains) > 0 || len(alt.Matches) > 0 || len(alt.Glob) > 0
}

func anyOfElementDeclares(alt AnyOfElement) bool {
	return alt.Equals != nil || alt.Contains != nil || alt.Matches != nil || alt.Glob != nil
}

func (s *searcher) fastMatchV2(query Query, stub *Stub) bool {
	return matchQueryHeaders(query, stub) && s.fastMatchBody(query, stub)
}

func matchQueryHeaders(query Query, stub *Stub) bool {
	if stub.Headers.Len() > 0 && len(query.Headers) == 0 {
		return false
	}

	if len(query.Headers) > 0 && !matchHeaders(query.Headers, stub.Headers) {
		return false
	}

	return true
}

func (s *searcher) fastMatchBody(query Query, stub *Stub) bool {
	matchers := stub.Matchers()

	if stub.DeclaresStreamMatchers() {
		if len(matchers) == 0 {
			return false
		}

		return s.fastMatchStream(query.Input, matchers)
	}

	unary := matchers[0]

	if len(query.Input) == 0 {
		return !inputHasConditions(unary)
	}

	if len(query.Input) == 1 {
		return s.fastMatchInput(query.Input[0], unary)
	}

	for _, v := range slices.Backward(query.Input) {
		if s.fastMatchInput(v, unary) {
			return true
		}
	}

	return false
}

func (s *searcher) fastRankV2(query Query, stub *Stub) float64 {
	if len(query.Headers) > 0 && !matchHeaders(query.Headers, stub.Headers) {
		return 0
	}

	headersRank := rankHeaders(query.Headers, stub.Headers)

	matchers := stub.Matchers()

	if stub.DeclaresStreamMatchers() {
		if len(matchers) == 0 {
			return headersRank
		}

		inputsBonus := 1000.0

		return headersRank + s.fastRankStream(query.Input, matchers) + inputsBonus
	}

	unary := matchers[0]

	if len(query.Input) == 0 {
		return headersRank
	}

	if len(query.Input) == 1 {
		return headersRank + s.fastRankInput(query.Input[0], unary)
	}

	n := len(query.Input)
	best := 0.0

	for i := n - 1; i >= 0; i-- {
		r := s.fastRankInput(query.Input[i], unary)
		if r > 0 {
			weighted := r * (float64(i+1) / float64(n))
			if weighted > best {
				best = weighted
			}
		}
	}

	return headersRank + best
}

//nolint:cyclop
func (s *searcher) fastMatchInput(queryData map[string]any, stubInput InputData) bool {
	if len(queryData) == 0 {
		return !inputHasConditions(stubInput)
	}

	if len(stubInput.AnyOf) == 0 && len(stubInput.Glob) == 0 {
		if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
			return equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder)
		}

		if len(stubInput.Contains) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Matches) == 0 {
			return contains(stubInput.Contains, queryData)
		}

		if len(stubInput.Matches) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 {
			return matches(stubInput.Matches, queryData)
		}
	}

	return matchInput(queryData, stubInput)
}

//nolint:cyclop
func (s *searcher) fastMatchStream(queryStream []map[string]any, stubStream []InputData) bool {
	declaresMatcher := func(e InputData) bool {
		return e.Equals != nil || e.Contains != nil || e.Matches != nil || e.Glob != nil ||
			slices.ContainsFunc(e.AnyOf, anyOfElementDeclares)
	}

	if !slices.ContainsFunc(stubStream, declaresMatcher) {
		return false
	}

	if len(queryStream) == 0 {
		return !slices.ContainsFunc(stubStream, inputHasConditions)
	}

	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastMatchInput(queryStream[0], stubStream[0])
	}

	if len(stubStream) == 1 && len(queryStream) > 1 {
		for i := range queryStream {
			if !s.fastMatchInput(queryStream[i], stubStream[0]) {
				return false
			}
		}

		return true
	}

	return matchStreamElements(queryStream, stubStream)
}

func (s *searcher) fastRankInput(queryData map[string]any, stubInput InputData) float64 {
	if len(queryData) == 0 {
		if !inputHasConditions(stubInput) {
			return 1.0
		}

		return 0
	}

	if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 &&
		len(stubInput.Glob) == 0 && len(stubInput.AnyOf) == 0 {
		if equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder) {
			return 1.0
		}

		return 0
	}

	return rankInput(queryData, stubInput)
}

func (s *searcher) fastRankStreamBroadcast(queryStream []map[string]any, pattern InputData) float64 {
	n := len(queryStream)

	var total float64

	for _, msg := range queryStream {
		total += s.fastRankInput(msg, pattern)
	}

	return total / float64(n)
}

func (s *searcher) fastRankStream(queryStream []map[string]any, stubStream []InputData) float64 {
	if len(queryStream) == 0 {
		if slices.ContainsFunc(stubStream, inputHasConditions) {
			return 0
		}

		return 1.0
	}

	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastRankInput(queryStream[0], stubStream[0])
	}

	if len(stubStream) == 1 && len(queryStream) > 1 {
		return s.fastRankStreamBroadcast(queryStream, stubStream[0])
	}

	return rankStreamElements(queryStream, stubStream)
}
