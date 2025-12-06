package deeply

import (
	"reflect"
)

// Equals checks if the expected and actual values are deeply equal.
// It returns true if any of the following conditions are met:
//   - The expected and actual values are both maps and have the same number of keys.
//   - The expected and actual values are both slices and have the same length.
//   - The expected and actual values are deeply equal using reflect.DeepEqual.
func Equals(expect, actual any) bool {
	return mapDeepCompare(expect, actual, Equals, func(left, right int) bool {
		return left == right
	}) || reflect.DeepEqual(expect, actual)
}

// EqualsIgnoreArrayOrder checks if the expected and actual values are deeply equal
// ignoring the order of arrays. It behaves similarly to Equals except that it
// uses slicesDeepEqualContains instead of slicesDeepEqual to compare slices.
func EqualsIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepCompare(expect, actual, EqualsIgnoreArrayOrder, func(left, right int) bool {
		return left == right
	}) ||
		slicesDeepCompare(expect, actual, EqualsIgnoreArrayOrder, func(aLen, bLen int) bool {
			return aLen == bLen
		}) ||
		reflect.DeepEqual(expect, actual)
}
