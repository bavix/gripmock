package stuber

func matchBidiStubMessage(stub *Stub, messageData map[string]any) bool {
	if stub.IsBidirectional() {
		if len(stub.Inputs) == 0 {
			return matchInputData(stub.Input, messageData)
		}

		return matchAnyStreamInput(stub.Inputs, messageData)
	}

	if stub.IsClientStream() {
		return matchAnyStreamInput(stub.Inputs, messageData)
	}

	if stub.IsUnary() || stub.IsServerStream() {
		return matchInputData(stub.Input, messageData)
	}

	return false
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
		if messageIndex < len(stub.Inputs) {
			return rankInputData(stub.Inputs[messageIndex], query.Input[0])
		}

		return 0.1 //nolint:mnd
	}

	return rankStub(query, stub)
}

func matchInputData(inputData InputData, messageData map[string]any) bool {
	if isInputMatcherEmpty(inputData) {
		return true
	}

	if !matchInputEquals(inputData.Equals, messageData) ||
		!matchInputContains(inputData.Contains, messageData) ||
		!matchInputRegex(inputData.Matches, messageData) {
		return false
	}

	if len(inputData.AnyOf) == 0 {
		return true
	}

	for i := range inputData.AnyOf {
		alt := &inputData.AnyOf[i]
		if matchInputEquals(alt.Equals, messageData) &&
			matchInputContains(alt.Contains, messageData) &&
			matchInputRegex(alt.Matches, messageData) {
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
		rankInputRegex(inputData.Matches, messageData)

	if len(inputData.AnyOf) == 0 {
		return base
	}

	bestAlt := 0.0

	for i := range inputData.AnyOf {
		alt := &inputData.AnyOf[i]

		r := rankInputEquals(alt.Equals, messageData) +
			rankInputContains(alt.Contains, messageData) +
			rankInputRegex(alt.Matches, messageData)

		if r > bestAlt {
			bestAlt = r
		}
	}

	return base + bestAlt
}

func matchInputEquals(expected map[string]any, messageData map[string]any) bool {
	for key, expectedValue := range expected {
		if actualValue, exists := findValueWithVariations(messageData, key); !exists || !deepEqual(actualValue, expectedValue) {
			return false
		}
	}

	return true
}

func matchInputContains(expected map[string]any, messageData map[string]any) bool {
	return matchInputByComparator(expected, messageData, contains)
}

func matchInputRegex(expected map[string]any, messageData map[string]any) bool {
	return matchInputByComparator(expected, messageData, matches)
}

func matchInputByComparator(
	expected map[string]any,
	messageData map[string]any,
	comparator func(map[string]any, any, bool) bool,
) bool {
	var scratch map[string]any

	for key, expectedValue := range expected {
		actualValue, exists := messageData[key]
		if !exists {
			return false
		}

		if scratch == nil {
			scratch = make(map[string]any, 1)
		}

		scratch[key] = expectedValue
		if !comparator(scratch, actualValue, false) {
			delete(scratch, key)

			return false
		}

		delete(scratch, key)
	}

	return true
}

func rankInputEquals(expected map[string]any, messageData map[string]any) float64 {
	total := 0.0

	for key, expectedValue := range expected {
		if actualValue, exists := findValueWithVariations(messageData, key); exists && deepEqual(actualValue, expectedValue) {
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
	comparator func(map[string]any, any, bool) bool,
) float64 {
	var scratch map[string]any

	total := 0.0

	for key, expectedValue := range expected {
		if actualValue, exists := messageData[key]; exists {
			if scratch == nil {
				scratch = make(map[string]any, 1)
			}

			scratch[key] = expectedValue
			if comparator(scratch, actualValue, false) {
				total += 10.0
			}

			delete(scratch, key)
		}
	}

	return total
}

func rankStub(query Query, stub *Stub) float64 {
	headersRank := rankHeaders(query.Headers, stub.Headers)

	if len(stub.Inputs) > 0 {
		return headersRank + rankStreamElements(query.Input, stub.Inputs)
	}

	if len(query.Input) == 1 {
		return headersRank + rankInput(query.Input[0], stub.Input)
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
	return len(inputData.Equals) == 0 && len(inputData.Contains) == 0 && len(inputData.Matches) == 0 && len(inputData.AnyOf) == 0
}
