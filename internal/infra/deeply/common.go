package deeply

import (
	"reflect"
)

type cmp func(expect, actual any) bool

func deepEqual(expect, actual any) bool {
	switch left := expect.(type) {
	case nil:
		return actual == nil
	case map[string]any:
		return mapAnyEqual(left, actual)
	case []any:
		return sliceAnyEqual(left, actual)
	}

	return reflect.DeepEqual(expect, actual)
}

func mapAnyEqual(expect map[string]any, actual any) bool {
	right, ok := actual.(map[string]any)
	if !ok || len(expect) != len(right) || (expect == nil) != (right == nil) {
		return false
	}

	for key, value := range expect {
		other, exists := right[key]
		if !exists || !deepEqual(value, other) {
			return false
		}
	}

	return true
}

func sliceAnyEqual(expect []any, actual any) bool {
	right, ok := actual.([]any)
	if !ok || len(expect) != len(right) || (expect == nil) != (right == nil) {
		return false
	}

	for i, value := range expect {
		if !deepEqual(value, right[i]) {
			return false
		}
	}

	return true
}

func structural(expect any) bool {
	switch expect.(type) {
	case map[string]any, []any:
		return true
	}

	kind := reflect.ValueOf(expect).Kind()

	return kind == reflect.Map || kind == reflect.Slice
}

func slicesDeepEqualContains(expect, actual reflect.Value, compare cmp) bool {
	marks := make([]bool, actual.Len())
	res := 0

	for i := range expect.Len() {
		for j := range actual.Len() {
			if !marks[j] && compare(expect.Index(i).Interface(), actual.Index(j).Interface()) {
				marks[j] = true
				res++

				break
			}
		}
	}

	return res == expect.Len()
}

func mapDeepEquals(expect, actual reflect.Value, compare cmp) bool {
	for iter := expect.MapRange(); iter.Next(); {
		value := actual.MapIndex(iter.Key())
		if value.Kind() == reflect.Invalid || !compare(iter.Value().Interface(), value.Interface()) {
			return false
		}
	}

	return true
}
