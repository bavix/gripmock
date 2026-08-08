package deeply

import (
	"reflect"
)

// Ranker is a function type used to rank matches between two values.
type ranker func(expect, actual any) float64

// RankMatch calculates a cumulative match score between expected and actual
// values, recursing into maps and slices and scoring other types directly.
func RankMatch(expected, actual any) float64 {
	return rankMatchWithDepth(expected, actual, 0)
}

const maxRankDepth = 64

func rankMatchWithDepth(expected, actual any, depth int) float64 {
	if depth > maxRankDepth {
		return 0
	}

	// An empty expected map matches anything weakly.
	if value, ok := expected.(map[string]any); ok && len(value) == 0 {
		return 0.1 //nolint:mnd
	}

	score := rank(expected, actual)

	score += slicesRankMatch(expected, actual, func(e, a any) float64 {
		return rankMatchWithDepth(e, a, depth+1)
	})

	score += mapRankMatch(expected, actual, func(e, a any) float64 {
		return rankMatchWithDepth(e, a, depth+1)
	})

	return score
}

// rank scores a match between two values. Non-strings compare by deep equality;
// strings compare exactly, then by regex (when the expected string contains
// metacharacters), then by normalized Levenshtein distance.
func rank(expect, actual any) float64 {
	if _, ok := actual.(bool); ok {
		return 0
	}

	var (
		expectedStr, expectedStringOk = expect.(string)
		actualStr, actualStringOk     = stringify(actual)
	)

	// Non-string values fall back to deep equality.
	if !expectedStringOk || !actualStringOk {
		if deepEqual(expect, actual) {
			return 1
		}

		return 0
	}

	if expectedStr == actualStr {
		return 1
	}

	if score, ok := regexScore(expectedStr, actualStr); ok {
		return score
	}

	// No regex match: score by Levenshtein distance.
	return distance(expectedStr, actualStr)
}

// regexScore scores how much of subject the pattern covers, or reports false
// when the pattern holds no metacharacters, cannot match, or fails to compile.
func regexScore(pattern, subject string) (float64, bool) {
	if !hasRegexMeta(pattern) || !canMatch(pattern, subject) {
		return 0, false
	}

	compiled, err := compileRegex(pattern)
	if err != nil {
		return 0, false
	}

	results := compiled.FindStringIndex(subject)
	if len(results) != 2 || len(subject) == 0 {
		return 0, false
	}

	return float64(results[1]-results[0]) / float64(len(subject)), true
}

// mapRankMatch scores a match between two maps by summing per-key compare scores
// over the keys both maps share, normalized by the larger key count.
func mapRankMatch(expect, actual any, compare ranker) float64 {
	if left, ok := expect.(map[string]any); ok {
		right, isMap := actual.(map[string]any)
		if !isMap {
			return 0
		}

		var res float64

		for key, value := range left {
			if other, exists := right[key]; exists {
				res += compare(value, other)
			}
		}

		return normalizeRank(res, max(len(left), len(right)))
	}

	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return 0
	}

	if reflect.TypeOf(expect) == nil {
		return 1
	}

	if reflect.TypeOf(expect).Kind() != reflect.Map {
		return 0
	}

	left := reflect.ValueOf(expect)
	right := reflect.ValueOf(actual)

	var res float64

	for iter := left.MapRange(); iter.Next(); {
		if other := right.MapIndex(iter.Key()); other.IsValid() {
			res += compare(iter.Value().Interface(), other.Interface())
		}
	}

	return normalizeRank(res, max(left.Len(), right.Len()))
}

func normalizeRank(score float64, total int) float64 {
	if total == 0 {
		return 1
	}

	return score / float64(total)
}

// slicesRankMatch scores a match between two slices: each expected element is
// greedily paired with the best unused actual element (compare), and the summed
// score is normalized by the longer length. Type-mismatched or non-slice inputs
// score 0; two nil inputs score 1.
//
//nolint:cyclop
func slicesRankMatch(expect, actual any, compare ranker) float64 {
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return 0
	}

	if reflect.TypeOf(expect) == nil {
		return 1
	}

	a := reflect.ValueOf(expect)
	b := reflect.ValueOf(actual)

	if a.Kind() != reflect.Slice || b.Kind() != reflect.Slice {
		return 0
	}

	var res float64

	// marked tracks right-slice indices already consumed by a left element, so
	// each right element is matched at most once.
	marked := make(map[int]struct{}, b.Len())

	for i := range a.Len() {
		for j := range b.Len() {
			if _, ok := marked[j]; ok {
				continue
			}

			if result := compare(a.Index(i).Interface(), b.Index(j).Interface()); result != 0 {
				res += result
				marked[j] = struct{}{}
			}
		}
	}

	total := max(a.Len(), b.Len())

	if res == 0 && total == 0 {
		return 1
	}

	return res / float64(total)
}

// distance returns the Levenshtein similarity of two strings, normalized by the
// longer string's length (1.0 = identical, 0.0 = fully different).
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

	for y := range column {
		column[y] = y
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
