package deeply

import (
	"reflect"
)

// cmp is a function type used to compare two values.
type cmp func(expect, actual any) bool

// slicesDeepEqualContains reports whether every expected element matches a
// distinct actual element under compare (each actual used at most once).
func slicesDeepEqualContains(expect, actual reflect.Value, compare cmp) bool {
	marks := make([]bool, actual.Len())
	res := 0

	for i := range expect.Len() {
		for j := range actual.Len() {
			if !marks[j] && compare(expect.Index(i).Interface(), actual.Index(j).Interface()) {
				marks[j] = true
				res++
			}
		}
	}

	return res == expect.Len()
}

// mapDeepEquals reports whether every expected key exists in actual with a
// matching value under compare.
func mapDeepEquals(expect, actual reflect.Value, compare cmp) bool {
	for _, v := range expect.MapKeys() {
		if actual.MapIndex(v).Kind() == reflect.Invalid ||
			!compare(expect.MapIndex(v).Interface(), actual.MapIndex(v).Interface()) {
			return false
		}
	}

	return true
}
