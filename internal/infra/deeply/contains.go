package deeply

import "reflect"

// Contains checks if the expected value is contained in the actual value.
// It returns true if any of the following conditions are met:
//   - The expected and actual values are deeply equal using reflect.DeepEqual.
//   - The expected and actual values are maps and all keys and values in the expected map
//     are contained in the actual map.
//   - The expected and actual values are slices and the expected slice is completely
//     contained in the actual slice.
func Contains(expect, actual any) bool {
	return mapDeepCompare(expect, actual, Contains, func(left, right int) bool {
		return left <= right
	}) || reflect.DeepEqual(expect, actual)
}

// ContainsIgnoreArrayOrder checks if the expected value is contained in the actual value.
// It returns true if any of the following conditions are met:
//   - The expected and actual values are deeply equal using reflect.DeepEqual.
//   - The expected and actual values are maps and all keys and values in the expected map
//     are contained in the actual map.
//   - The expected and actual values are slices and the expected slice is partially
//     contained in the actual slice. The order of elements in the slice is not important.
func ContainsIgnoreArrayOrder(expect, actual any) bool {
	return mapDeepCompare(expect, actual, ContainsIgnoreArrayOrder, func(left, right int) bool {
		return left <= right
	}) ||
		slicesDeepCompare(expect, actual, ContainsIgnoreArrayOrder, func(aLen, bLen int) bool {
			return aLen <= bLen
		}) ||
		reflect.DeepEqual(expect, actual)
}
