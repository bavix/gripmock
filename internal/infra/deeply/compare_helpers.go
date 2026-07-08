package deeply

import "reflect"

// mapDeepCompare is a shared map comparator with customizable length rule.
func mapDeepCompare(expect, actual any, compare cmp, lengthOK func(left, right int) bool) bool {
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
