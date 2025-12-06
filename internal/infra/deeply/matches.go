package deeply

import (
	"log"
	"reflect"
	"regexp"

	"github.com/spf13/cast"
)

// Matches checks if the expected and actual values match.
// It returns true if any of the following conditions are met:
//   - The expected and actual values are both nil.
//   - The expected and actual values are both slices and have the same length.
//   - The expected and actual values are both maps and have the same number of keys.
//   - The expected and actual values match using a regular expression.
//   - The expected and actual values are deeply equal using reflect.DeepEqual.
func Matches(expect, actual any) bool {
	return mapDeepMatches(expect, actual, Matches) ||
		slicesDeepMatches(expect, actual, Matches) ||
		regexMatch(expect, actual) ||
		reflect.DeepEqual(expect, actual)
}

// MatchesIgnoreArrayOrder checks if the expected and actual values match
// ignoring the order of arrays. It behaves similarly to Matches except that it
// uses slicesDeepContains instead of slicesDeepMatches to compare slices.
func MatchesIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepMatches(expect, actual, MatchesIgnoreArrayOrder) ||
		slicesDeepMatchesIgnoreOrder(expect, actual, MatchesIgnoreArrayOrder) ||
		regexMatch(expect, actual) ||
		reflect.DeepEqual(expect, actual)
}

// slicesDeepMatches checks if the expected and actual slices match.
// It returns true if the expected and actual values are both slices and have
// the same length.
func slicesDeepMatches(expect, actual any, compare cmp) bool {
	return slicesDeepCompareOrdered(expect, actual, compare)
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

// regexMatch checks if the expected regular expression matches the actual string.
// It returns true if the regular expression matches the string, false otherwise.
//
// Parameters:
// expect: The expected regular expression. This can be of any type, but it is
//
//	converted to a string before being used as the regular expression.
//
// actual: The actual string to be matched. This should be a string, but it is
//
//	first converted to a string before being matched.
//
// Returns:
// A boolean value indicating whether the regular expression matches the string.
// If there is an error converting the expected or actual values to strings, or if
// there is an error matching the regular expression with the string, the function
// logs the error and returns false.
func regexMatch(expect, actual any) bool {
	// If actual is a boolean, return false.
	if _, ok := actual.(bool); ok {
		return false
	}

	// Convert the expected and actual values to string.
	var (
		expectedStr, expectedStringOk = expect.(string)        // Expected regular expression as a string.
		actualStr, actualStringErr    = cast.ToStringE(actual) // Actual string to be matched.
	)

	// If the values are not string, return false.
	if !expectedStringOk || actualStringErr != nil {
		return false
	}

	// Match the regular expression with the string.
	match, err := regexp.MatchString(expectedStr, actualStr)
	if err != nil {
		// If there is an error matching the regular expression with the string,
		// log the error and return false.
		log.Printf("Error on matching regex %s with %s error:%v\n", expect, actual, err)

		return false
	}

	// Return the result of the match.
	return match
}
