package deeply

import (
	"reflect"
	"regexp"

	"github.com/spf13/cast"
)

// Ranker is a function type used to rank matches between two values.
type ranker func(expect, actual any) float64

// RankMatch calculates a match score between expected and actual values.
//
// This function uses recursive matching for maps and slices and assesses
// the match for other types. The final score is the cumulative result of
// matches for maps, slices, and other values.
//
// Parameters:
//   - expected: The expected value.
//   - actual: The actual value.
//
// Returns:
//   - A float64 representing the cumulative match score.
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
//
// It compares two strings and returns a float64 representing the match score.
// The function first checks if the actual value is a boolean and returns 0 if it is.
// Then it converts the expected and actual values to strings. If the values are not
// strings or if there is an error converting them to strings, the function checks
// if the values are deeply equal and returns the corresponding match score.
// If the strings are equal, the function returns the full match score.
// Next, the function tries to compile the expected string as a regular expression
// and finds the first match in the actual string. If a match is found, the function
// calculates the match score based on the length of the match. If no match is found,
// the function calculates the match score based on the Levenshtein distance
// between the two strings.
//
// Parameters:
// - expect: The expected string.
// - actual: The actual string.
//
// Returns:
// - The match score between the expected and actual strings.
func rank(expect, actual any) float64 {
	// Check if the actual value is a boolean and return 0 if it is.
	if _, ok := actual.(bool); ok {
		return 0
	}

	// Convert the expected and actual values to strings.
	var (
		expectedStr, expectedStringOk = expect.(string)
		actualStr, actualStringErr    = cast.ToStringE(actual)
	)

	// If the values are not strings or if there is an error converting them to strings,
	// check if the values are deeply equal and return the corresponding match score.
	if !expectedStringOk || actualStringErr != nil {
		if reflect.DeepEqual(expect, actual) {
			return 1 // Full match.
		}

		return 0 // No match.
	}

	// If the strings are equal, return the full match score.
	if expectedStr == actualStr {
		return 1
	}

	// Try to compile the expected string as a regular expression and find the
	// first match in the actual string. If a match is found, calculate the match
	// score based on the length of the match.
	compile, err := regexp.Compile(expectedStr)
	if compile != nil && err == nil {
		results := compile.FindStringIndex(actualStr)

		// If a match is found, calculate the match score based on the length of
		// the match.
		if len(results) == 2 && len(actualStr) > 0 {
			return float64(results[1]-results[0]) / float64(len(actualStr))
		}
	}

	// If no match is found, calculate the match score based on the Levenshtein
	// distance between the two strings.
	return distance(expectedStr, actualStr)
}

// mapRankMatch calculates the match score between two maps.
//
// It iterates over the keys of the left map and finds the corresponding key in
// the right map. If a match is found, it calculates the match score between
// the values of the keys and adds it to the total score. It marks the keys
// that have been matched to avoid duplicate matches. The function returns the
// total score divided by the maximum number of keys in the two maps.
//
// Parameters:
//   - expect: The expected map.
//   - actual: The actual map.
//   - compare: The ranker function used to compare values.
//
// Returns:
//   - The match score between the expected and actual maps.
//
//nolint:cyclop
func mapRankMatch(expect, actual any, compare ranker) float64 {
	// Check if the types of the expected and actual values are the same.
	// If they are not, return 0.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return 0
	}

	// Check if the types of the expected and actual values are nil.
	// If they are, return 1.
	if reflect.TypeOf(expect) == nil {
		return 1
	}

	// Check if the expected value is a map.
	// If it is not, return 0.
	if reflect.TypeOf(expect).Kind() != reflect.Map {
		return 0
	}

	// Convert the expected and actual values to reflect.Value.
	left := reflect.ValueOf(expect)
	right := reflect.ValueOf(actual)

	// Initialize the total score.
	var res float64

	// Calculate the maximum number of keys in the two maps.
	total := max(left.Len(), right.Len())

	// Create a map to keep track of the keys that have been matched.
	marked := make(map[reflect.Value]bool, total)

	// Iterate over the keys of the left map.
	for _, k := range left.MapKeys() {
		// If the corresponding key exists in the right map, calculate the match
		// score between the values and add it to the total score.
		// Mark the key as matched.
		if right.MapIndex(k).IsValid() {
			res += compare(left.MapIndex(k).Interface(), right.MapIndex(k).Interface())
			marked[right.MapIndex(k)] = true
		}
	}

	// Iterate over the keys of the right map.
	// If a key has not been marked as matched, calculate the match score between
	// the corresponding values in the left and right maps and add it to the total
	// score.
	for _, k := range right.MapKeys() {
		if _, ok := marked[k]; ok {
			continue
		}

		if left.MapIndex(k).IsValid() {
			res += compare(left.MapIndex(k).Interface(), right.MapIndex(k).Interface())
		}
	}

	// If the total score is 0 and the maximum number of keys is 0, return 1.
	if res == 0 && total == 0 {
		return 1
	}

	// Return the total score divided by the maximum number of keys.
	return res / float64(total)
}

// slicesRankMatch is a function that calculates the match score between two
// slices or maps. It takes the expected and actual values and a ranker
// function that compares two values and returns a match score between 0 and 1.
//
// The ranker function is called for each pair of values in the slices or maps,
// and the match scores are accumulated. The function returns the accumulated
// match score divided by the maximum number of values in the slices or maps.
//
// If the types of the expected and actual values are not equal, the function
// returns 0. If either the expected or actual value is nil, the function
// returns 1. If the types of the expected and actual values are not slice or
// map, the function returns 0.
//
// The ranker function is called for each pair of values in the slices or maps,
// and the match scores are accumulated. The function returns the accumulated
// match score divided by the maximum number of values in the slices or maps.
//
// The function uses a marked algorithm to avoid redundant comparisons.
//
//nolint:cyclop
func slicesRankMatch(expect, actual any, compare ranker) float64 {
	// Check if the types of the expected and actual values are equal.
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return 0
	}

	// If both values are nil, return 1.
	if reflect.TypeOf(expect) == nil {
		return 1
	}

	// Convert the expected and actual values to reflect.Value.
	a := reflect.ValueOf(expect)
	b := reflect.ValueOf(actual)

	// If the types of the expected and actual values are not slice, return 0.
	if a.Kind() != reflect.Slice || b.Kind() != reflect.Slice {
		return 0
	}

	var res float64 // Initialize the total score.

	marked := make(map[int]struct{}, b.Len()) // Create a map to keep track of the keys that have been marked as matched.

	// Iterate over the values of the left slice.
	for i := range a.Len() {
		// Iterate over the values of the right slice.
		for j := range b.Len() {
			// Skip the value if it has already been marked as matched.
			if _, ok := marked[j]; ok {
				continue
			}

			// Calculate the match score between the values of the indices and
			// add it to the total score if the result is not 0.
			if result := compare(a.Index(i).Interface(), b.Index(j).Interface()); result != 0 {
				res += result
				marked[j] = struct{}{}
			}
		}
	}

	total := max(a.Len(), b.Len()) // Calculate the maximum number of values in the two slices.

	// If the total score is 0 and the maximum number of values is 0, return 1.
	if res == 0 && total == 0 {
		return 1
	}

	// Return the total score divided by the maximum number of values.
	return res / float64(total)
}

// distance calculates the Levenshtein distance between two strings.
// It returns a float64 representing the distance normalized by the length of the
// longer string.
//
// The Levenshtein distance is a measure of the number of single-character edits
// needed to transform one string into another, such as insertion, deletion, or
// substitution.
//
// Parameters:
// - s: The first string.
// - t: The second string.
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

// ASCII-optimized version with stack-allocated buffer.
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
