package deeply

import (
	"reflect"
)

// cmp is a function type used to compare two values.
type cmp func(expect, actual any) bool

// slicesDeepEqualContains checks if the expected slice contains all the values of the actual slice.
// It returns true if any of the following conditions are met:
//   - The expected and actual values are both nil.
//   - The expected and actual values are both slices and have the same length.
//   - The expected and actual values are deeply equal using the provided compare function.
func slicesDeepEqualContains(expect, actual reflect.Value, compare cmp) bool {
	marks := make([]bool, actual.Len()) // Create a map to keep track of the keys that have been marked as matched.
	res := 0                            // Initialize the total number of matched values.

	// Iterate over the values of the expected slice.
	for i := range expect.Len() {
		// Iterate over the values of the actual slice.
		for j := range actual.Len() {
			// Skip the value if it has already been marked as matched.
			if !marks[j] && compare(expect.Index(i).Interface(), actual.Index(j).Interface()) {
				marks[j] = true // Mark the value as matched.
				res++           // Increment the total number of matched values.
			}
		}
	}

	// Return true if the total number of matched values is equal to the length of the expected slice.
	return res == expect.Len()
}

// mapDeepEquals checks if the expected and actual values are deeply equal as maps.
// It returns true if any of the following conditions are met:
//   - The expected and actual values are both nil.
//   - The expected and actual values are both maps and have the same number of keys.
//   - The expected and actual values are deeply equal using the provided compare function.
func mapDeepEquals(expect, actual reflect.Value, compare cmp) bool {
	// Iterate over the keys of the expected map.
	for _, v := range expect.MapKeys() {
		// Check if the actual value has a corresponding key and if the values are deeply equal.
		if actual.MapIndex(v).Kind() == reflect.Invalid ||
			!compare(expect.MapIndex(v).Interface(), actual.MapIndex(v).Interface()) {
			return false
		}
	}

	// Return true if all values are deeply equal.
	return true
}
