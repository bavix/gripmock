package deeply

import (
	"reflect"
)

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
