package deeply

import (
	"log"
	"reflect"
	"regexp"

	"github.com/spf13/cast"
)

// MatchesIgnoreArrayOrder checks if the expected and actual values match
// ignoring the order of arrays. It behaves similarly to Matches except that it
// uses slicesDeepContains instead of slicesDeepMatches to compare slices.
func MatchesIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepMatches(expect, actual, MatchesIgnoreArrayOrder) ||
		slicesDeepMatchesIgnoreOrder(expect, actual, MatchesIgnoreArrayOrder) ||
		regexMatch(expect, actual) ||
		reflect.DeepEqual(expect, actual)
}

// slicesDeepMatchesIgnoreOrder compares slices allowing actual to be same length or longer.
func slicesDeepMatchesIgnoreOrder(expect, actual any, compare cmp) bool {
	return slicesDeepCompare(expect, actual, compare, func(aLen, bLen int) bool {
		return aLen == bLen
	})
}

// mapDeepMatches checks if the expected and actual maps match.
// It returns true if the expected and actual values are both maps and have
// the same number of keys.
func mapDeepMatches(expect, actual any, compare cmp) bool {
	return mapDeepCompare(expect, actual, compare, func(left, right int) bool {
		return left <= right
	})
}

// regexMatch reports whether the expected string, used as a regular expression,
// matches the actual value (stringified). Non-string inputs or a bad pattern
// return false; a pattern error is logged.
func regexMatch(expect, actual any) bool {
	if _, ok := actual.(bool); ok {
		return false
	}

	var (
		expectedStr, expectedStringOk = expect.(string)
		actualStr, actualStringErr    = cast.ToStringE(actual)
	)

	if !expectedStringOk || actualStringErr != nil {
		return false
	}

	match, err := regexp.MatchString(expectedStr, actualStr)
	if err != nil {
		log.Printf("Error on matching regex %s with %s error:%v\n", expect, actual, err)

		return false
	}

	return match
}
