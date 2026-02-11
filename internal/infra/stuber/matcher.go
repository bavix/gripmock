package stuber

import (
	"encoding/json"
	"log"
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

// Get retrieves a compiled regex from cache or compiles it if not found.
func getRegex(pattern string) (*regexp.Regexp, error) {
	if regexCache != nil {
		if re, exists := regexCache.Get(pattern); exists {
			return re, nil
		}
	}

	// Compile and cache (or just compile if cache init failed)
	re, err := regexp.Compile(pattern)
	if err == nil && regexCache != nil {
		regexCache.Add(pattern, re)
	}

	return re, err
}

// getRegexCacheStats returns regex cache statistics (length, capacity).
//
//nolint:unparam // capacity is a constant for API compatibility
func getRegexCacheStats() (int, int) {
	if regexCache == nil {
		return 0, regexCacheSize
	}

	return regexCache.Len(), regexCacheSize // Fixed capacity
}

// clearRegexCache clears the regex cache.
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
func matchHeaders(queryHeaders map[string]any, stubHeaders InputHeader) bool {
	return equals(stubHeaders.Equals, queryHeaders, false) &&
		contains(stubHeaders.Contains, queryHeaders, false) &&
		matches(stubHeaders.Matches, queryHeaders, false)
}

// matchInput checks if query data matches stub input.
func matchInput(queryData map[string]any, stubInput InputData) bool {
	return equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder) &&
		contains(stubInput.Contains, queryData, stubInput.IgnoreArrayOrder) &&
		matches(stubInput.Matches, queryData, stubInput.IgnoreArrayOrder)
}

// rankHeaders ranks query headers against stub headers.
func rankHeaders(queryHeaders map[string]any, stubHeaders InputHeader) float64 {
	if stubHeaders.Len() == 0 {
		return 0
	}

	return deeply.RankMatch(stubHeaders.Equals, queryHeaders) +
		deeply.RankMatch(stubHeaders.Contains, queryHeaders) +
		deeply.RankMatch(stubHeaders.Matches, queryHeaders)
}

// rankInput ranks query data against stub input.
func rankInput(queryData map[string]any, stubInput InputData) float64 {
	return deeply.RankMatch(stubInput.Equals, queryData) +
		deeply.RankMatch(stubInput.Contains, queryData) +
		deeply.RankMatch(stubInput.Matches, queryData)
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

	return ultraFastSpecializedEquals(expected, actual)
}

// ultraFastSpecializedEquals provides ultra-fast comparison for common types without reflect.
//
//nolint:cyclop,funlen,gocognit,gocyclo
func ultraFastSpecializedEquals(expected, actual any) bool {
	// Ultra-fast path: same type comparison (most common case)
	//nolint:nestif
	if reflect.TypeOf(expected) == reflect.TypeOf(actual) {
		switch e := expected.(type) {
		case string:
			if a, ok := actual.(string); ok {
				return e == a
			}
		case int:
			if a, ok := actual.(int); ok {
				return e == a
			}
		case float64:
			if a, ok := actual.(float64); ok {
				return e == a
			}
		case bool:
			if a, ok := actual.(bool); ok {
				return e == a
			}
		case int64:
			if a, ok := actual.(int64); ok {
				return e == a
			}
		}
	}

	// Fast path: number type conversions (common case)
	switch e := expected.(type) {
	case int:
		switch a := actual.(type) {
		case int:
			return e == a
		case float64:
			return float64(e) == a
		case int64:
			return int64(e) == a
		}
	case float64:
		switch a := actual.(type) {
		case float64:
			return e == a
		case int:
			return e == float64(a)
		case int64:
			return e == float64(a)
		case json.Number:
			if aFloat, err := a.Float64(); err == nil {
				return e == aFloat
			}
		}
	case int64:
		switch a := actual.(type) {
		case int64:
			return e == a
		case float64:
			return float64(e) == a
		case int:
			return e == int64(a)
		}
	case string:
		if a, ok := actual.(string); ok {
			return e == a
		}
	case bool:
		if a, ok := actual.(bool); ok {
			return e == a
		}
	case json.Number:
		if a, ok := actual.(json.Number); ok {
			return e.String() == a.String()
		}

		if aFloat, err := e.Float64(); err == nil {
			switch a := actual.(type) {
			case float64:
				return aFloat == a
			case int:
				return aFloat == float64(a)
			case int64:
				return aFloat == float64(a)
			}
		}
	}

	// Fallback to reflect for complex types (rare case)
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

// matchStreamElements checks if the query stream matches the stub stream.
//
//nolint:cyclop
func matchStreamElements(queryStream []map[string]any, stubStream []InputData) bool {
	// For client streaming, grpctestify sends an extra empty message at the end
	// We need to handle this case by checking if the last message is empty
	effectiveQueryLength := len(queryStream)
	if effectiveQueryLength > 0 {
		lastMessage := queryStream[effectiveQueryLength-1]
		if len(lastMessage) == 0 {
			effectiveQueryLength--
		}
	}

	// Enforce exact length match for client streaming to avoid out-of-range panics
	if effectiveQueryLength != len(stubStream) {
		return false
	}

	// For client streaming, allow partial matching for ranking purposes
	// Length mismatch is handled in ranking function

	// STRICT: If query stream is empty but stub expects data, no match
	if effectiveQueryLength == 0 && len(stubStream) > 0 {
		return false
	}

	for i := range effectiveQueryLength {
		queryItem := queryStream[i]
		stubItem := stubStream[i]

		// Check if this stub item has any matchers defined
		hasMatchers := len(stubItem.Equals) > 0 || len(stubItem.Contains) > 0 || len(stubItem.Matches) > 0
		if !hasMatchers {
			return false
		}

		// Check equals matcher
		if len(stubItem.Equals) > 0 {
			if !equals(stubItem.Equals, queryItem, stubItem.IgnoreArrayOrder) {
				return false
			}
		}

		// Check contains matcher
		if len(stubItem.Contains) > 0 {
			if !contains(stubItem.Contains, queryItem, stubItem.IgnoreArrayOrder) {
				return false
			}
		}

		// Check matches matcher
		if len(stubItem.Matches) > 0 {
			if !matches(stubItem.Matches, queryItem, stubItem.IgnoreArrayOrder) {
				return false
			}
		}
	}

	return true
}

// rankStreamElements ranks the match between query stream and stub stream.
func rankStreamElements(queryStream []map[string]any, stubStream []InputData) float64 {
	effectiveQueryLength := getEffectiveQueryLength(queryStream)
	if effectiveQueryLength != len(stubStream) {
		return 0
	}

	if effectiveQueryLength == 0 && len(stubStream) > 0 {
		return 0
	}

	totalRank, perfectMatches := computeStreamElementRanks(queryStream, stubStream, effectiveQueryLength)
	lengthBonus := float64(effectiveQueryLength) * 10.0   //nolint:mnd
	perfectMatchBonus := float64(perfectMatches) * 1000.0 //nolint:mnd

	completeMatchBonus := 0.0
	if perfectMatches == effectiveQueryLength && effectiveQueryLength > 0 {
		completeMatchBonus = 10000.0
	}

	specificityBonus := countStreamMatchers(stubStream) * 50.0 //nolint:mnd

	return totalRank + lengthBonus + perfectMatchBonus + completeMatchBonus + specificityBonus
}

func getEffectiveQueryLength(queryStream []map[string]any) int {
	n := len(queryStream)
	if n > 0 && len(queryStream[n-1]) == 0 {
		return n - 1
	}

	return n
}

func computeStreamElementRanks(
	queryStream []map[string]any,
	stubStream []InputData,
	effectiveQueryLength int,
) (float64, int) {
	var totalRank float64

	var perfectMatches int

	for i := range effectiveQueryLength {
		queryItem := queryStream[i]
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
