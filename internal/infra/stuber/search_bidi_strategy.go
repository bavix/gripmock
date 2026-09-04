package stuber

func matchBidiStubHeaders(stub *Stub, queryHeaders map[string]any) bool {
	if stub.Headers.Len() > 0 && len(queryHeaders) == 0 {
		return false
	}

	if len(queryHeaders) > 0 && !matchHeaders(queryHeaders, stub.Headers) {
		return false
	}

	return true
}

func HandlerCandidate(stub *Stub, query QueryBidi) bool {
	if stub.Handler == nil {
		return false
	}

	if stub.Session != "" && stub.Session != query.Session {
		return false
	}

	return matchBidiStubHeaders(stub, normalizeHeaderKeys(query.Headers))
}

func matchBidiStubMessage(stub *Stub, queryHeaders map[string]any, messageData map[string]any) bool {
	if !matchBidiStubHeaders(stub, queryHeaders) {
		return false
	}

	matchers := stub.Matchers()

	if stub.DeclaresStreamMatchers() {
		return matchAnyStreamInput(matchers, messageData)
	}

	return matchInputData(matchers[0], messageData)
}

func matchAnyStreamInput(inputs []InputData, messageData map[string]any) bool {
	for _, streamElement := range inputs {
		if matchInputData(streamElement, messageData) {
			return true
		}
	}

	return false
}

func scoreBidiStubMessage(query Query, stub *Stub, messageIndex int) float64 {
	if stub.IsBidirectional() && len(stub.Inputs) > 0 {
		headersRank := rankHeaders(query.Headers, stub.Headers)

		if messageIndex < len(stub.Inputs) {
			return headersRank + rankInputData(stub.Inputs[messageIndex], query.Input[0])
		}

		return headersRank + 0.1 //nolint:mnd
	}

	return rankStub(query, stub)
}

//nolint:cyclop
func matchInputData(inputData InputData, messageData map[string]any) bool {
	if isInputMatcherEmpty(inputData) {
		return true
	}

	if !matchInputEquals(inputData.Equals, messageData) ||
		!matchInputContains(inputData.Contains, messageData) ||
		!matchInputRegex(inputData.Matches, messageData) ||
		!matchInputGlob(inputData.Glob, messageData) {
		return false
	}

	if len(inputData.AnyOf) == 0 {
		return true
	}

	for i := range inputData.AnyOf {
		alt := &inputData.AnyOf[i]
		if matchInputEquals(alt.Equals, messageData) &&
			matchInputContains(alt.Contains, messageData) &&
			matchInputRegex(alt.Matches, messageData) &&
			matchInputGlob(alt.Glob, messageData) {
			return true
		}
	}

	return false
}

func rankInputData(inputData InputData, messageData map[string]any) float64 {
	if isInputMatcherEmpty(inputData) {
		return 1.0
	}

	base := rankInputEquals(inputData.Equals, messageData) +
		rankInputContains(inputData.Contains, messageData) +
		rankInputRegex(inputData.Matches, messageData) +
		rankGlob(inputData.Glob, messageData)

	if len(inputData.AnyOf) == 0 {
		return base
	}

	bestAlt := 0.0

	for i := range inputData.AnyOf {
		alt := &inputData.AnyOf[i]

		r := rankInputEquals(alt.Equals, messageData) +
			rankInputContains(alt.Contains, messageData) +
			rankInputRegex(alt.Matches, messageData) +
			rankGlob(alt.Glob, messageData)

		if r > bestAlt {
			bestAlt = r
		}
	}

	return base + bestAlt
}

func matchInputEquals(expected map[string]any, messageData map[string]any) bool {
	return equals(expected, messageData, false)
}

func matchInputContains(expected map[string]any, messageData map[string]any) bool {
	return contains(expected, messageData)
}

func matchInputRegex(expected map[string]any, messageData map[string]any) bool {
	return matches(expected, messageData)
}

func matchInputGlob(expected map[string]any, messageData map[string]any) bool {
	return globMatch(expected, messageData)
}

func rankInputEquals(expected map[string]any, messageData map[string]any) float64 {
	total := 0.0

	for key, expectedValue := range expected {
		if actualValue, exists := findValueWithVariations(messageData, key); exists && compareFieldValue(expectedValue, actualValue, false) {
			total += 100.0
		}
	}

	return total
}

func rankInputContains(expected map[string]any, messageData map[string]any) float64 {
	return rankInputByComparator(expected, messageData, contains)
}

func rankInputRegex(expected map[string]any, messageData map[string]any) float64 {
	return rankInputByComparator(expected, messageData, matches)
}

func rankInputByComparator(
	expected map[string]any,
	messageData map[string]any,
	comparator func(map[string]any, any) bool,
) float64 {
	total := 0.0

	for key, expectedValue := range expected {
		actualValue, exists := messageData[key]
		if !exists {
			continue
		}

		if comparator(map[string]any{key: expectedValue}, map[string]any{key: actualValue}) {
			total += 10.0
		}
	}

	return total
}

func rankStub(query Query, stub *Stub) float64 {
	headersRank := rankHeaders(query.Headers, stub.Headers)

	matchers := stub.Matchers()

	if stub.DeclaresStreamMatchers() && len(matchers) > 0 {
		return headersRank + rankStreamElements(query.Input, matchers)
	}

	if !stub.DeclaresStreamMatchers() && len(query.Input) == 1 {
		return headersRank + rankInput(query.Input[0], matchers[0])
	}

	return headersRank
}

func findValueWithVariations(messageData map[string]any, key string) (any, bool) {
	if value, exists := messageData[key]; exists {
		return value, true
	}

	hasUnderscore, hasUpper := keyStyleFlags(key)

	if hasUnderscore {
		if value, exists := messageData[toCamelCase(key)]; exists {
			return value, true
		}
	}

	if hasUpper {
		if value, exists := messageData[toSnakeCase(key)]; exists {
			return value, true
		}
	}

	return nil, false
}

func keyStyleFlags(s string) (bool, bool) {
	hasUnderscore := false
	hasUpper := false

	for i := range len(s) {
		if s[i] == '_' {
			hasUnderscore = true
		}

		if s[i] >= 'A' && s[i] <= 'Z' {
			hasUpper = true
		}

		if hasUnderscore && hasUpper {
			return true, true
		}
	}

	return hasUnderscore, hasUpper
}

func isInputMatcherEmpty(inputData InputData) bool {
	return len(inputData.Equals) == 0 &&
		len(inputData.Contains) == 0 &&
		len(inputData.Matches) == 0 &&
		len(inputData.Glob) == 0 &&
		len(inputData.AnyOf) == 0
}
