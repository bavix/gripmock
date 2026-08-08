package deeply

import (
	"fmt"
	"log"
	"reflect"

	"github.com/spf13/cast"
)

// MatchesIgnoreArrayOrder checks if the expected and actual values match
// ignoring the order of arrays. It behaves similarly to Matches except that it
// uses slicesDeepContains instead of slicesDeepMatches to compare slices.
func MatchesIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepMatches(expect, actual, MatchesIgnoreArrayOrder) ||
		slicesDeepMatchesIgnoreOrder(expect, actual, MatchesIgnoreArrayOrder) ||
		regexMatch(expect, actual) ||
		(!structural(expect) && reflect.DeepEqual(expect, actual))
}

func stringify(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case nil:
		return "", true
	case bool:
		return cast.ToString(v), true
	case []byte:
		return string(v), true
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return cast.ToString(v), true
	case fmt.Stringer:
		return v.String(), true
	case error:
		return v.Error(), true
	}

	if reflected := reflect.ValueOf(value); reflected.Kind() == reflect.String {
		return reflected.String(), true
	}

	return "", false
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
	if _, isBool := actual.(bool); isBool {
		return false
	}

	expectedStr, ok := expect.(string)
	if !ok {
		return false
	}

	actualStr, ok := stringify(actual)
	if !ok {
		return false
	}

	if !canMatch(expectedStr, actualStr) {
		return false
	}

	compiled, err := compileRegex(expectedStr)
	if err != nil {
		log.Printf("Error on matching regex %s with %s error:%v\n", expect, actual, err)

		return false
	}

	match := compiled.MatchString(actualStr)

	return match
}
