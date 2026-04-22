package stuber

import "github.com/bavix/gripmock/v3/internal/infra/protoconv"

// calcSpecificity calculates the specificity score for a stub against a query.
// Higher specificity means more fields match between stub and query.
// Headers are given higher weight to ensure stubs with headers are preferred.
func (s *searcher) calcSpecificity(stub *Stub, query Query) int {
	// Specificity now reflects only input structure, header impact is accounted in rank via rankHeaders
	specificity := 0

	if len(query.Input) == 0 {
		return specificity
	}

	// Priority to Inputs (newer functionality) over Input (legacy)
	if len(stub.Inputs) > 0 {
		return specificity + s.calcSpecificityStream(stub.Inputs, query.Input)
	}

	if len(query.Input) == 1 {
		return specificity + s.calcSpecificityUnary(stub.Input, query.Input[0])
	}

	return specificity
}

// calcSpecificityUnary calculates specificity for unary case.
func (s *searcher) calcSpecificityUnary(stubInput InputData, queryData map[string]any) int {
	fieldExistsWithNonDefaultValue := func(stubKey string) bool {
		queryValue, found := findValueWithVariations(queryData, stubKey)
		if !found {
			return false
		}

		return !protoconv.IsDefaultValue(queryValue)
	}

	specificity := countMatcherKeys(stubInput.Equals, fieldExistsWithNonDefaultValue)
	specificity += countMatcherKeys(stubInput.Contains, fieldExistsWithNonDefaultValue)
	specificity += countMatcherKeys(stubInput.Matches, fieldExistsWithNonDefaultValue)

	for _, alt := range stubInput.AnyOf {
		specificity += countMatcherKeys(alt.Equals, fieldExistsWithNonDefaultValue)
		specificity += countMatcherKeys(alt.Contains, fieldExistsWithNonDefaultValue)
		specificity += countMatcherKeys(alt.Matches, fieldExistsWithNonDefaultValue)
	}

	return specificity
}

func countMatcherKeys(m map[string]any, predicate func(string) bool) int {
	n := 0

	for key := range m {
		if predicate(key) {
			n++
		}
	}

	return n
}

// calcSpecificityStream calculates specificity for stream case.
func (s *searcher) calcSpecificityStream(stubStream []InputData, queryStream []map[string]any) int {
	if len(stubStream) == 0 || len(queryStream) == 0 {
		return 0
	}

	totalSpecificity := 0

	minLen := min(len(queryStream), len(stubStream))

	for i := range minLen {
		totalSpecificity += s.calcSpecificityUnary(stubStream[i], queryStream[i])
	}

	return totalSpecificity
}
