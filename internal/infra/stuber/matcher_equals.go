package stuber

import (
	"encoding/json"
	"reflect"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

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
		actualValue, exists := findValueWithVariations(actualMap, key)
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
