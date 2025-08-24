package matcher

import (
	"reflect"
	"regexp"
	"sort"
)

// The package implements unified matching semantics for v4 matchers.
// Rules:
// - equals/contains/matches are combined with logical AND
// - any is a logical OR of nested matchers
// - ignoreArrayOrder toggles order-insensitive comparison for arrays

type (
	// Matcher mirrors domain Matcher to avoid import cycles in infra layer.
	Matcher struct {
		Equals           map[string]any    `json:"equals,omitempty"`
		Contains         map[string]any    `json:"contains,omitempty"`
		Matches          map[string]string `json:"matches,omitempty"`
		Any              []Matcher         `json:"any,omitempty"`
		IgnoreArrayOrder bool              `json:"ignoreArrayOrder,omitempty"`
	}
)

// Match returns true when candidate satisfies the provided matcher.

func Match(m Matcher, candidate map[string]any) bool {
	if len(m.Any) > 0 {
		if !matchAny(m.Any, candidate) {
			return false
		}
	}

	if len(m.Equals) > 0 && !deepSubsetEqual(candidate, m.Equals, m.IgnoreArrayOrder) {
		return false
	}

	if len(m.Contains) > 0 && !deepContains(candidate, m.Contains, m.IgnoreArrayOrder) {
		return false
	}

	if len(m.Matches) > 0 && !regexMatchAll(candidate, m.Matches) {
		return false
	}

	return true
}

func matchAny(alternatives []Matcher, candidate map[string]any) bool {
	for _, alt := range alternatives {
		if Match(alt, candidate) {
			return true
		}
	}

	return false
}

// deepSubsetEqual ensures that for every key in expected (recursively),
// the candidate has the same value. Extra fields in candidate are ignored.
//
//nolint:cyclop
func deepSubsetEqual(candidate any, expected any, ignoreArrayOrder bool) bool {
	switch ev := expected.(type) {
	case map[string]any:
		cm, ok := candidate.(map[string]any)
		if !ok {
			return false
		}

		for k, v := range ev {
			if !deepSubsetEqual(cm[k], v, ignoreArrayOrder) {
				return false
			}
		}

		return true
	case []any:
		ca, ok := candidate.([]any)
		if !ok {
			return false
		}

		if !ignoreArrayOrder {
			if len(ca) != len(ev) {
				return false
			}

			for i := range ev {
				if !deepSubsetEqual(ca[i], ev[i], ignoreArrayOrder) {
					return false
				}
			}

			return true
		}

		return equalAsSets(ca, ev, ignoreArrayOrder)
	default:
		return reflect.DeepEqual(normalizeScalar(candidate), normalizeScalar(expected))
	}
}

// deepContains ensures that candidate contains expected. For strings it
// performs substring checks. For arrays it checks that all expected elements
// are present in candidate (order-insensitive when flag is set).
//
//nolint:cyclop
func deepContains(candidate any, expected any, ignoreArrayOrder bool) bool {
	switch ev := expected.(type) {
	case map[string]any:
		cm, ok := candidate.(map[string]any)
		if !ok {
			return false
		}

		for k, v := range ev {
			if !deepContains(cm[k], v, ignoreArrayOrder) {
				return false
			}
		}

		return true
	case []any:
		ca, ok := candidate.([]any)
		if !ok {
			return false
		}
		// required: every ev element must be contained in ca
		used := make([]bool, len(ca))

		for _, want := range ev {
			found := false

			for j := range ca {
				if used[j] {
					continue
				}

				if deepSubsetEqual(ca[j], want, ignoreArrayOrder) {
					used[j] = true
					found = true

					break
				}
			}

			if !found {
				return false
			}
		}

		return true
	case string:
		cs, ok := candidate.(string)
		if !ok {
			return false
		}

		return containsString(cs, ev)
	default:
		return reflect.DeepEqual(normalizeScalar(candidate), normalizeScalar(expected))
	}
}

func regexMatchAll(candidate map[string]any, patterns map[string]string) bool {
	for key, pat := range patterns {
		v, ok := candidate[key]
		if !ok {
			return false
		}

		s, ok := v.(string)
		if !ok {
			return false
		}

		rx, err := regexp.Compile(pat)
		if err != nil {
			return false
		}

		if !rx.MatchString(s) {
			return false
		}
	}

	return true
}

// equalAsSets compares two slices as sets. Elements are compared using
// deepSubsetEqual semantics to support nested structures.
func equalAsSets(ca, ev []any, ignoreArrayOrder bool) bool {
	if len(ca) != len(ev) {
		return false
	}

	// We build canonical string representations for sorting stable pairs.
	used := make([]bool, len(ca))

	for _, want := range ev {
		found := false

		for j := range ca {
			if used[j] {
				continue
			}

			if deepSubsetEqual(ca[j], want, ignoreArrayOrder) {
				used[j] = true
				found = true

				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func containsString(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

// indexOf is a small helper to avoid importing strings. It returns the first
// index of substr in s or -1 when not found.
func indexOf(s, substr string) int {
	// Use a simple sliding window; adequate for matching small payloads.
	if len(substr) == 0 {
		return 0
	}

	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// normalizeScalar coerces numeric types to a common comparable form.
//
//nolint:cyclop
func normalizeScalar(v any) any {
	switch x := v.(type) {
	case float32:
		return float64(x)
	case float64:
		return x
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case uint:
		return uint64(x)
	case uint8:
		return uint64(x)
	case uint16:
		return uint64(x)
	case uint32:
		return uint64(x)
	case uint64:
		return x
	default:
		return v
	}
}

// SortKeys returns sorted keys of a map. This helper is currently unused but
// kept for potential deterministic operations.
func SortKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// distance calculates the Levenshtein distance similarity between two strings.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func distance(s, t string) float64 {
	// Fast path for identical strings
	if s == t {
		return 1.0
	}

	// Fast path for empty strings
	lenS, lenT := len(s), len(t)
	if lenS == 0 || lenT == 0 {
		return 0.0
	}

	// ASCII fast path optimization (common case)
	if isASCII(s) && isASCII(t) {
		return distanceASCII(s, t)
	}

	// Unicode path for non-ASCII strings
	r1, r2 := []rune(s), []rune(t)
	len1, len2 := len(r1), len(r2)

	column := make([]int, len1+1)
	for y := range column {
		column[y] = y
	}

	for x := 1; x <= len2; x++ {
		r2char := r2[x-1]
		column[0] = x
		lastDiag := x - 1

		for y := 1; y <= len1; y++ {
			oldDiag := column[y]

			cost := 0
			if r1[y-1] != r2char {
				cost = 1
			}

			column[y] = min(
				column[y]+1, // deletion
				min(column[y-1]+1, // insertion
					lastDiag+cost), // substitution
			)

			lastDiag = oldDiag
		}
	}

	maxLength := max(len1, len2)

	return (float64(maxLength) - float64(column[len1])) / float64(maxLength)
}

// distanceASCII calculates Levenshtein distance for ASCII strings with stack optimization.
func distanceASCII(s, t string) float64 {
	lenS, lenT := len(s), len(t)

	// Use stack allocation for small strings (common case)
	const maxStackLen = 64

	var (
		columnStack [maxStackLen]int
		column      []int
	)

	if lenS+1 <= maxStackLen {
		column = columnStack[:lenS+1]
	} else {
		column = make([]int, lenS+1)
	}

	// Initialize column using copy for small sizes
	if lenS+1 <= maxStackLen {
		copy(column, columnStack[:])
	} else {
		for y := range column {
			column[y] = y
		}
	}

	for x := 1; x <= lenT; x++ {
		tChar := t[x-1]
		column[0] = x
		lastDiag := x - 1

		for y := 1; y <= lenS; y++ {
			oldDiag := column[y]

			cost := 0
			if s[y-1] != tChar {
				cost = 1
			}

			column[y] = min(
				column[y]+1,
				min(column[y-1]+1, lastDiag+cost),
			)

			lastDiag = oldDiag
		}
	}

	maxLength := max(lenS, lenT)

	return (float64(maxLength) - float64(column[lenS])) / float64(maxLength)
}

// isASCII checks if a string contains only ASCII characters.
func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= 128 { //nolint:mnd
			return false
		}
	}

	return true
}

// Contains checks if the expected value is contained in the actual value.
func Contains(expect, actual any) bool {
	return mapDeepContains(expect, actual, Contains) || reflect.DeepEqual(expect, actual)
}

// ContainsIgnoreArrayOrder checks if the expected value is contained in the actual value.
func ContainsIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepContains(expect, actual, ContainsIgnoreArrayOrder) ||
		slicesDeepContains(expect, actual, ContainsIgnoreArrayOrder) ||
		reflect.DeepEqual(expect, actual)
}

// cmp is a function type for comparison operations.
type cmp func(expect, actual any) bool

// mapDeepContains checks if the expected map is contained in the actual map.
func mapDeepContains(expect, actual any, compare cmp) bool {
	// Check if the types are the same.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	// Check if the expected value is a map.
	expectMap, ok := expect.(map[string]any)
	if !ok {
		return false
	}

	// Check if the actual value is a map.
	actualMap, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	// Check if all keys and values in the expected map are contained in the actual map.
	for key, value := range expectMap {
		if !compare(actualMap[key], value) {
			return false
		}
	}

	return true
}

// slicesDeepContains checks if the expected slice is contained in the actual slice.
func slicesDeepContains(expect, actual any, compare cmp) bool {
	// Check if the types are the same.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	// Check if the expected value is a slice.
	expectSlice, ok := expect.([]any)
	if !ok {
		return false
	}

	// Check if the actual value is a slice.
	actualSlice, ok := actual.([]any)
	if !ok {
		return false
	}

	// Check if all elements in the expected slice are contained in the actual slice.
	for _, expectElement := range expectSlice {
		found := false

		for _, actualElement := range actualSlice {
			if compare(actualElement, expectElement) {
				found = true

				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// Equals checks if the expected and actual values are deeply equal.
func Equals(expect, actual any) bool {
	return mapDeepEqual(expect, actual, Equals) || reflect.DeepEqual(expect, actual)
}

// EqualsIgnoreArrayOrder checks if the expected and actual values are deeply equal
// ignoring the order of arrays.
func EqualsIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepEqual(expect, actual, EqualsIgnoreArrayOrder) ||
		slicesDeepEqual(expect, actual, EqualsIgnoreArrayOrder) ||
		reflect.DeepEqual(expect, actual)
}

// mapDeepEqual checks if the expected and actual values are deeply equal as maps.
func mapDeepEqual(expect, actual any, compare cmp) bool {
	// Check if the types are the same.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	// Check if the expected value is a map.
	expectMap, ok := expect.(map[string]any)
	if !ok {
		return false
	}

	// Check if the actual value is a map.
	actualMap, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	// Check if all keys and values in the expected map are equal to the actual map.
	for key, value := range expectMap {
		if !compare(actualMap[key], value) {
			return false
		}
	}

	return true
}

// slicesDeepEqual checks if the expected and actual values are deeply equal as slices.
func slicesDeepEqual(expect, actual any, compare cmp) bool {
	// Check if the types are the same.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	// Check if the expected value is a slice.
	expectSlice, ok := expect.([]any)
	if !ok {
		return false
	}

	// Check if the actual value is a slice.
	actualSlice, ok := actual.([]any)
	if !ok {
		return false
	}

	// Check if all elements in the expected slice are equal to the actual slice.
	if len(expectSlice) != len(actualSlice) {
		return false
	}

	for i, expectElement := range expectSlice {
		if !compare(actualSlice[i], expectElement) {
			return false
		}
	}

	return true
}

// Matches checks if the expected and actual values match using regex patterns.
func Matches(expect, actual any) bool {
	return mapDeepMatches(expect, actual, Matches) ||
		slicesDeepMatches(expect, actual, Matches) ||
		regexMatch(expect, actual) ||
		reflect.DeepEqual(expect, actual)
}

// MatchesIgnoreArrayOrder checks if the expected and actual values match
// ignoring the order of arrays.
func MatchesIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepMatches(expect, actual, MatchesIgnoreArrayOrder) ||
		slicesDeepContains(expect, actual, MatchesIgnoreArrayOrder) ||
		regexMatch(expect, actual) ||
		reflect.DeepEqual(expect, actual)
}

// mapDeepMatches checks if the expected and actual values match as maps.
func mapDeepMatches(expect, actual any, compare cmp) bool {
	// Check if the types are the same.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	// Check if the expected value is a map.
	expectMap, ok := expect.(map[string]any)
	if !ok {
		return false
	}

	// Check if the actual value is a map.
	actualMap, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	// Check if all keys and values in the expected map match the actual map.
	for key, value := range expectMap {
		if !compare(actualMap[key], value) {
			return false
		}
	}

	return true
}

// slicesDeepMatches checks if the expected and actual values match as slices.
func slicesDeepMatches(expect, actual any, compare cmp) bool {
	// Check if the types are the same.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	// Check if the expected value is a slice.
	expectSlice, ok := expect.([]any)
	if !ok {
		return false
	}

	// Check if the actual value is a slice.
	actualSlice, ok := actual.([]any)
	if !ok {
		return false
	}

	// Check if all elements in the expected slice match the actual slice.
	if len(expectSlice) != len(actualSlice) {
		return false
	}

	for i, expectElement := range expectSlice {
		if !compare(actualSlice[i], expectElement) {
			return false
		}
	}

	return true
}

// regexMatch checks if the expected and actual values match using regex patterns.
func regexMatch(expect, actual any) bool {
	// Check if the expected value is a map with regex patterns.
	expectMap, ok := expect.(map[string]any)
	if !ok {
		return false
	}

	// Check if the actual value is a map.
	actualMap, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	// Check if all keys and values in the expected map match the actual map using regex.
	for key, pattern := range expectMap {
		patternStr, ok := pattern.(string)
		if !ok {
			return false
		}

		actualValue, exists := actualMap[key]
		if !exists {
			return false
		}

		actualStr, ok := actualValue.(string)
		if !ok {
			return false
		}

		matched, err := regexp.MatchString(patternStr, actualStr)
		if err != nil || !matched {
			return false
		}
	}

	return true
}

// RankMatch calculates a match score between expected and actual values.
func RankMatch(expected, actual any) float64 {
	// Special case handling for empty maps.
	if value, ok := expected.(map[string]any); ok && len(value) == 0 {
		return 0.1 //nolint:mnd
	}

	// Calculate the match score for non-collection types.
	score := rank(expected, actual)

	// Include scores from slice comparisons.
	score += slicesRankMatch(expected, actual, RankMatch)

	// Include scores from map comparisons.
	score += mapRankMatch(expected, actual, RankMatch)

	// Return the total match score.
	return score
}

// rank is a function that ranks the matches between two strings.
func rank(expected, actual any) float64 {
	// Check if the actual value is a boolean and return 0 if it is.
	if _, ok := actual.(bool); ok {
		return 0
	}

	// Convert expected and actual to strings.
	expectedStr, ok1 := expected.(string)
	actualStr, ok2 := actual.(string)

	if !ok1 || !ok2 {
		// If conversion fails, check if values are deeply equal.
		if reflect.DeepEqual(expected, actual) {
			return 1.0
		}

		return 0.0
	}

	// Calculate string similarity using distance function.
	return distance(expectedStr, actualStr)
}

// slicesRankMatch calculates match scores for slice comparisons.
func slicesRankMatch(expected, actual any, rankFunc func(any, any) float64) float64 {
	// Check if both values are slices.
	expectedSlice, ok1 := expected.([]any)
	actualSlice, ok2 := actual.([]any)

	if !ok1 || !ok2 {
		return 0.0
	}

	// Calculate average score for matching elements.
	totalScore := 0.0
	matchCount := 0

	for _, exp := range expectedSlice {
		for _, act := range actualSlice {
			score := rankFunc(exp, act)
			if score > 0 {
				totalScore += score
				matchCount++
			}
		}
	}

	if matchCount == 0 {
		return 0.0
	}

	return totalScore / float64(matchCount)
}

// mapRankMatch calculates match scores for map comparisons.
func mapRankMatch(expected, actual any, rankFunc func(any, any) float64) float64 {
	// Check if both values are maps.
	expectedMap, ok1 := expected.(map[string]any)
	actualMap, ok2 := actual.(map[string]any)

	if !ok1 || !ok2 {
		return 0.0
	}

	// Calculate average score for matching keys and values.
	totalScore := 0.0
	matchCount := 0

	for key, expValue := range expectedMap {
		if actValue, exists := actualMap[key]; exists {
			score := rankFunc(expValue, actValue)
			if score > 0 {
				totalScore += score
				matchCount++
			}
		}
	}

	if matchCount == 0 {
		return 0.0
	}

	return totalScore / float64(matchCount)
}
