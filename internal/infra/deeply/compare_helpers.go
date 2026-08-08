package deeply

import "reflect"

// mapDeepCompare is a shared map comparator with customizable length rule.
func mapDeepCompare(expect, actual any, compare cmp, lengthOK func(left, right int) bool) bool {
	if left, ok := expect.(map[string]any); ok {
		return mapAnyCompare(left, actual, compare, lengthOK)
	}

	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	if reflect.TypeOf(expect) == nil {
		return true
	}

	if reflect.TypeOf(expect).Kind() != reflect.Map {
		return false
	}

	left := reflect.ValueOf(expect)
	right := reflect.ValueOf(actual)

	if !lengthOK(left.Len(), right.Len()) {
		return false
	}

	return mapDeepEquals(left, right, compare)
}

func mapAnyCompare(expect map[string]any, actual any, compare cmp, lengthOK func(left, right int) bool) bool {
	right, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	if !lengthOK(len(expect), len(right)) {
		return false
	}

	for key, value := range expect {
		other, exists := right[key]
		if !exists || !compare(value, other) {
			return false
		}
	}

	return true
}

// slicesDeepCompare is a shared slice comparator with customizable length rule.
func slicesDeepCompare(expect, actual any, compare cmp, lengthOK func(aLen, bLen int) bool) bool {
	if reflect.TypeOf(expect) != reflect.TypeOf(actual) {
		return false
	}

	if reflect.TypeOf(expect) == nil {
		return true
	}

	if reflect.TypeOf(expect).Kind() != reflect.Slice {
		return false
	}

	a := reflect.ValueOf(expect)
	b := reflect.ValueOf(actual)

	if !lengthOK(a.Len(), b.Len()) {
		return false
	}

	return slicesDeepEqualContains(a, b, compare)
}
