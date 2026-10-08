package deeply

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
)

func MatchesIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepMatches(expect, actual, MatchesIgnoreArrayOrder) ||
		slicesDeepMatchesIgnoreOrder(expect, actual, MatchesIgnoreArrayOrder) ||
		regexMatch(expect, actual) ||
		(!structural(expect) && reflect.DeepEqual(expect, actual))
}

// Stringify renders a scalar value the way regexp matching sees it.
func Stringify(value any) (string, bool) {
	return stringify(value)
}

func stringify(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case nil:
		return "", true
	case bool:
		return strconv.FormatBool(v), true
	case []byte:
		return string(v), true
	case fmt.Stringer:
		return v.String(), true
	case error:
		return v.Error(), true
	}

	if s, ok := stringifyNumber(value); ok {
		return s, true
	}

	if reflected := reflect.ValueOf(value); reflected.Kind() == reflect.String {
		return reflected.String(), true
	}

	return "", false
}

//nolint:cyclop // one branch per numeric kind; a table would not be clearer.
func stringifyNumber(value any) (string, bool) {
	switch v := value.(type) {
	case int:
		return strconv.FormatInt(int64(v), 10), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		return "", false
	}
}

func slicesDeepMatchesIgnoreOrder(expect, actual any, compare cmp) bool {
	return slicesDeepCompare(expect, actual, compare, func(aLen, bLen int) bool {
		return aLen == bLen
	})
}

func mapDeepMatches(expect, actual any, compare cmp) bool {
	return mapDeepCompare(expect, actual, compare, func(left, right int) bool {
		return left <= right
	})
}

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
