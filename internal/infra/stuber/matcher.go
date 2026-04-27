package stuber

import (
	"encoding/json"
	"errors"
	"log"
	"path"
	"reflect"
	"regexp"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

const (
	// regexCacheSize is the maximum number of regex patterns to cache.
	regexCacheSize = 1000
)

// Global LRU cache for regex patterns with size limit.
//
//nolint:gochecknoglobals
var regexCache *lru.Cache[string, *regexp.Regexp]

//nolint:gochecknoinits
func init() {
	var err error
	// Create LRU cache with size limit of regexCacheSize regex patterns
	regexCache, err = lru.New[string, *regexp.Regexp](regexCacheSize)
	if err != nil {
		log.Printf("[gripmock] failed to create regex cache: %v", err)

		regexCache = nil
	}
}

// clearRegexCache clears the regex cache (for testing).
func clearRegexCache() {
	if regexCache != nil {
		regexCache.Purge()
	}
}

// match checks if a given query matches a given stub.
//
// It checks if the query matches the stub's input data and headers using
// the equals, contains, and matches methods.
func match(query Query, stub *Stub) bool {
	// Check headers first
	if !matchHeaders(query.Headers, stub.Headers) {
		return false
	}

	// Check if the query's input data matches the stub's input data
	return matchInput(query.Data(), stub.Input)
}

// matchHeaders checks if query headers match stub headers.
//
//nolint:cyclop
func matchHeaders(queryHeaders map[string]any, stubHeaders InputHeader) bool {
	if !equals(stubHeaders.Equals, queryHeaders, false) ||
		!contains(stubHeaders.Contains, queryHeaders, false) ||
		!matches(stubHeaders.Matches, queryHeaders, false) ||
		!globMatch(stubHeaders.Glob, queryHeaders) {
		return false
	}

	if len(stubHeaders.AnyOf) == 0 {
		return true
	}

	for i := range stubHeaders.AnyOf {
		alt := &stubHeaders.AnyOf[i]
		if equals(alt.Equals, queryHeaders, false) &&
			contains(alt.Contains, queryHeaders, false) &&
			matches(alt.Matches, queryHeaders, false) &&
			globMatch(alt.Glob, queryHeaders) {
			return true
		}
	}

	return false
}

// matchInput checks if query data matches stub input.
//
//nolint:cyclop
func matchInput(queryData map[string]any, stubInput InputData) bool {
	if !equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder) ||
		!contains(stubInput.Contains, queryData, stubInput.IgnoreArrayOrder) ||
		!matches(stubInput.Matches, queryData, stubInput.IgnoreArrayOrder) ||
		!globMatch(stubInput.Glob, queryData) {
		return false
	}

	if len(stubInput.AnyOf) == 0 {
		return true
	}

	for i := range stubInput.AnyOf {
		alt := &stubInput.AnyOf[i]
		if equals(alt.Equals, queryData, alt.IgnoreArrayOrder) &&
			contains(alt.Contains, queryData, alt.IgnoreArrayOrder) &&
			matches(alt.Matches, queryData, alt.IgnoreArrayOrder) &&
			globMatch(alt.Glob, queryData) {
			return true
		}
	}

	return false
}

// rankHeaders ranks query headers against stub headers.
func rankHeaders(queryHeaders map[string]any, stubHeaders InputHeader) float64 {
	if stubHeaders.Len() == 0 {
		return 0
	}

	base := deeply.RankMatch(stubHeaders.Equals, queryHeaders) +
		deeply.RankMatch(stubHeaders.Contains, queryHeaders) +
		deeply.RankMatch(stubHeaders.Matches, queryHeaders) +
		rankGlob(stubHeaders.Glob, queryHeaders)

	if len(stubHeaders.AnyOf) == 0 {
		return base
	}

	best := 0.0

	for i := range stubHeaders.AnyOf {
		alt := &stubHeaders.AnyOf[i]

		r := deeply.RankMatch(alt.Equals, queryHeaders) +
			deeply.RankMatch(alt.Contains, queryHeaders) +
			deeply.RankMatch(alt.Matches, queryHeaders) +
			rankGlob(alt.Glob, queryHeaders)
		if r > best {
			best = r
		}
	}

	return base + best
}

// rankInput ranks query data against stub input.
func rankInput(queryData map[string]any, stubInput InputData) float64 {
	base := deeply.RankMatch(stubInput.Equals, queryData) +
		deeply.RankMatch(stubInput.Contains, queryData) +
		deeply.RankMatch(stubInput.Matches, queryData) +
		rankGlob(stubInput.Glob, queryData)

	if len(stubInput.AnyOf) == 0 {
		return base
	}

	best := 0.0

	for i := range stubInput.AnyOf {
		alt := &stubInput.AnyOf[i]

		r := deeply.RankMatch(alt.Equals, queryData) +
			deeply.RankMatch(alt.Contains, queryData) +
			deeply.RankMatch(alt.Matches, queryData) +
			rankGlob(alt.Glob, queryData)
		if r > best {
			best = r
		}
	}

	return base + best
}

// equals compares two values for deep equality.
func equals(expected map[string]any, actual any, orderIgnore bool) bool {
	if len(expected) == 0 {
		return true
	}

	actualMap, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	for key, expectedValue := range expected {
		actualValue, exists := actualMap[key]
		if !exists {
			return false
		}

		if !compareFieldValue(expectedValue, actualValue, orderIgnore) {
			return false
		}
	}

	return true
}

func compareFieldValue(expected, actual any, orderIgnore bool) bool {
	if orderIgnore {
		if eSlice, eOk := expected.([]any); eOk {
			if aSlice, aOk := actual.([]any); aOk {
				return deeply.EqualsIgnoreArrayOrder(eSlice, aSlice)
			}
		}
	}

	return fieldValueEquals(expected, actual)
}

// toFloat64 converts common numeric types to float64 for comparison.
func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case json.Number:
		f, err := x.Float64()

		return f, err == nil
	default:
		return 0, false
	}
}

// sameTypeEq is a generic helper for comparing values of the same comparable type.
func sameTypeEq[T comparable](e, a T) bool {
	return e == a
}

// sameTypeValueEquals handles comparison when both values have the same type.
// Second return is true if the type was handled (no need to fall through).
//
//nolint:cyclop
func sameTypeValueEquals(expected, actual any) (bool, bool) {
	switch e := expected.(type) {
	case string:
		a, ok := actual.(string)

		return ok && sameTypeEq(e, a), true
	case int:
		a, ok := actual.(int)

		return ok && sameTypeEq(e, a), true
	case float64:
		a, ok := actual.(float64)

		return ok && sameTypeEq(e, a), true
	case bool:
		a, ok := actual.(bool)

		return ok && sameTypeEq(e, a), true
	case int64:
		a, ok := actual.(int64)

		return ok && sameTypeEq(e, a), true
	case json.Number:
		a, ok := actual.(json.Number)

		return ok && e.String() == a.String(), true
	default:
		return false, false
	}
}

// fieldValueEquals compares two values with fast paths for common types.
func fieldValueEquals(expected, actual any) bool {
	if reflect.TypeOf(expected) == reflect.TypeOf(actual) {
		if eq, handled := sameTypeValueEquals(expected, actual); handled {
			return eq
		}
	}

	if eF, eOk := toFloat64(expected); eOk {
		if aF, aOk := toFloat64(actual); aOk {
			return eF == aF
		}
	}

	switch e := expected.(type) {
	case string:
		if a, ok := actual.(string); ok {
			return e == a
		}
	case bool:
		if a, ok := actual.(bool); ok {
			return e == a
		}
	}

	return reflect.DeepEqual(expected, actual)
}

// contains checks if the expected map is a subset of the actual value.
//
// It returns true if the expected map is a subset of the actual value,
// otherwise false.
func contains(expected map[string]any, actual any, _ bool) bool {
	if len(expected) == 0 {
		return true
	}

	return deeply.ContainsIgnoreArrayOrder(expected, actual)
}

// matches checks if the expected map matches the actual value using regular expressions.
//
// It returns true if the expected map matches the actual value using regular expressions,
// otherwise false.
func matches(expected map[string]any, actual any, _ bool) bool {
	if len(expected) == 0 {
		return true
	}

	return deeply.MatchesIgnoreArrayOrder(expected, actual)
}

// globMatch checks if the expected map matches the actual value using glob patterns.
//
// It returns true if all glob patterns match, otherwise false.
// Supports nested map traversal for matching at any depth.
func globMatch(expected map[string]any, actual any) bool {
	if len(expected) == 0 {
		return true
	}

	actualMap, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	for key, pattern := range expected {
		actualValue, exists := actualMap[key]
		if !exists {
			return false
		}

		if err := matchGlobValue(pattern, actualValue); err != nil {
			return false
		}
	}

	return true
}

// matchGlobValue matches a pattern against a value, supporting nested maps.
func matchGlobValue(pattern, actual any) error {
	patternStr, isStringPattern := pattern.(string)
	actualStr, isStringActual := actual.(string)

	if isStringPattern && isStringActual {
		matched, err := path.Match(patternStr, actualStr)
		if err != nil || !matched {
			return errFail
		}

		return nil
	}

	patternMap, isPatternMap := pattern.(map[string]any)
	actualMap, isActualMap := actual.(map[string]any)

	if isPatternMap && isActualMap {
		for key, pat := range patternMap {
			act, exists := actualMap[key]
			if !exists {
				return errFail
			}

			if err := matchGlobValue(pat, act); err != nil {
				return err
			}
		}

		return nil
	}

	return errFail
}

var errFail = errors.New("glob match failed")

// rankGlob calculates rank for glob pattern matching.
func rankGlob(expected map[string]any, actual any) float64 {
	if len(expected) == 0 {
		return 0
	}

	return rankGlobValue(expected, actual)
}

func rankGlobValue(pattern, actual any) float64 {
	patternStr, isStringPattern := pattern.(string)
	actualStr, isStringActual := actual.(string)

	if isStringPattern && isStringActual {
		matched, err := path.Match(patternStr, actualStr)
		if err == nil && matched {
			return 1.0
		}

		return 0
	}

	patternMap, isPatternMap := pattern.(map[string]any)
	actualMap, isActualMap := actual.(map[string]any)

	if isPatternMap && isActualMap {
		var rank float64

		for key, pat := range patternMap {
			act, exists := actualMap[key]
			if !exists {
				continue
			}

			rank += rankGlobValue(pat, act)
		}

		return rank
	}

	return 0
}

// streamItemMatches checks if a single query item matches the stub item matchers.
//
//nolint:cyclop
func streamItemMatches(stubItem InputData, queryItem map[string]any) bool {
	if len(stubItem.Equals) == 0 && len(stubItem.Contains) == 0 && len(stubItem.Matches) == 0 && len(stubItem.Glob) == 0 {
		return false
	}

	return (len(stubItem.Equals) == 0 || equals(stubItem.Equals, queryItem, stubItem.IgnoreArrayOrder)) &&
		(len(stubItem.Contains) == 0 || contains(stubItem.Contains, queryItem, stubItem.IgnoreArrayOrder)) &&
		(len(stubItem.Matches) == 0 || matches(stubItem.Matches, queryItem, stubItem.IgnoreArrayOrder)) &&
		(len(stubItem.Glob) == 0 || globMatch(stubItem.Glob, queryItem))
}

// matchStreamElements checks if the query stream matches the stub stream.
func matchStreamElements(queryStream []map[string]any, stubStream []InputData) bool {
	n := len(queryStream)

	// STRICT: If query stream is empty but stub expects data, no match
	if n == 0 && len(stubStream) > 0 {
		return false
	}

	// Broadcast mode: stub has a single pattern — every message must match it.
	// This covers the case where a stub uses inputs:[{pattern}] to match any
	// number of streaming messages (e.g. PROD_789 stub with 2-message stream).
	if len(stubStream) == 1 && n > 1 {
		pattern := stubStream[0]

		for _, msg := range queryStream {
			if !streamItemMatches(pattern, msg) {
				return false
			}
		}

		return true
	}

	if n != len(stubStream) {
		return false
	}

	for i, msg := range queryStream {
		if !streamItemMatches(stubStream[i], msg) {
			return false
		}
	}

	return true
}

// rankStreamElements ranks the match between query stream and stub stream.
func rankStreamElements(queryStream []map[string]any, stubStream []InputData) float64 {
	n := len(queryStream)

	if n == 0 && len(stubStream) > 0 {
		return 0
	}

	// Broadcast mode: single pattern matched against all messages.
	if len(stubStream) == 1 && n > 1 {
		totalRank, perfectMatches := computeStreamElementRanksBroadcast(queryStream, stubStream[0])
		avgRank := totalRank / float64(n)
		perfectMatchBonus := float64(perfectMatches) * 1000.0      //nolint:mnd
		specificityBonus := countStreamMatchers(stubStream) * 50.0 //nolint:mnd

		return avgRank + perfectMatchBonus + specificityBonus
	}

	if n != len(stubStream) {
		return 0
	}

	totalRank, perfectMatches := computeStreamElementRanks(queryStream, stubStream)
	lengthBonus := float64(n) * 10.0                      //nolint:mnd
	perfectMatchBonus := float64(perfectMatches) * 1000.0 //nolint:mnd

	completeMatchBonus := 0.0
	if perfectMatches == n && n > 0 {
		completeMatchBonus = 10000.0
	}

	specificityBonus := countStreamMatchers(stubStream) * 50.0 //nolint:mnd

	return totalRank + lengthBonus + perfectMatchBonus + completeMatchBonus + specificityBonus
}

func computeStreamElementRanksBroadcast(
	queryStream []map[string]any,
	stubPattern InputData,
) (float64, int) {
	var totalRank float64

	var perfectMatches int

	for _, queryItem := range queryStream {
		equalsRank := 0.0
		if len(stubPattern.Equals) > 0 && equals(stubPattern.Equals, queryItem, stubPattern.IgnoreArrayOrder) {
			equalsRank = 1.0
		}

		containsRank := deeply.RankMatch(stubPattern.Contains, queryItem)
		matchesRank := deeply.RankMatch(stubPattern.Matches, queryItem)
		elementRank := equalsRank*100.0 + containsRank*0.1 + matchesRank*0.1 //nolint:mnd
		totalRank += elementRank

		if equalsRank > 0.99 { //nolint:mnd
			perfectMatches++
		}
	}

	return totalRank, perfectMatches
}

func computeStreamElementRanks(
	queryStream []map[string]any,
	stubStream []InputData,
) (float64, int) {
	var totalRank float64

	var perfectMatches int

	for i, queryItem := range queryStream {
		stubItem := stubStream[i]

		equalsRank := 0.0
		if len(stubItem.Equals) > 0 && equals(stubItem.Equals, queryItem, stubItem.IgnoreArrayOrder) {
			equalsRank = 1.0
		}

		containsRank := deeply.RankMatch(stubItem.Contains, queryItem)
		matchesRank := deeply.RankMatch(stubItem.Matches, queryItem)
		elementRank := equalsRank*100.0 + containsRank*0.1 + matchesRank*0.1 //nolint:mnd
		totalRank += elementRank

		if equalsRank > 0.99 { //nolint:mnd
			perfectMatches++
		}
	}

	return totalRank, perfectMatches
}

func countStreamMatchers(stubStream []InputData) float64 {
	var total int

	for _, stubItem := range stubStream {
		total += countNonNil(stubItem.Equals) + countNonNil(stubItem.Contains) + countNonNil(stubItem.Matches)
	}

	return float64(total)
}

func countNonNil(m map[string]any) int {
	var n int

	for _, v := range m {
		if v != nil {
			n++
		}
	}

	return n
}
