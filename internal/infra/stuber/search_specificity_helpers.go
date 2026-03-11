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
// Counts the number of fields that exist in both stub and query.
// Supports all field types: Equals, Contains, and Matches.
// Supports field name variations (camelCase vs snake_case).
// Excludes fields with default/empty values from specificity calculation.
//
// Parameters:
// - stubInput: The stub's input data
// - queryData: The query's input data
//
// Returns:
// - int: The number of matching fields (excluding default values).
func (s *searcher) calcSpecificityUnary(stubInput InputData, queryData map[string]any) int {
	specificity := 0

	// Helper function to check if field exists with variations and has non-default value
	fieldExistsWithNonDefaultValue := func(stubKey string) bool {
		// Try to find the field in query with variations
		queryValue, found := findValueWithVariations(queryData, stubKey)
		if !found {
			return false
		}
		// Check if query value is non-default
		return !protoconv.IsDefaultValue(queryValue)
	}

	// Count equals fields
	for key := range stubInput.Equals {
		if fieldExistsWithNonDefaultValue(key) {
			specificity++
		}
	}

	// Count contains fields
	for key := range stubInput.Contains {
		if fieldExistsWithNonDefaultValue(key) {
			specificity++
		}
	}

	// Count matches fields
	for key := range stubInput.Matches {
		if fieldExistsWithNonDefaultValue(key) {
			specificity++
		}
	}

	return specificity
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
