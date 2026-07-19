package plugins

import (
	"cmp"
	"fmt"
)

func arrayFuncs() map[string]any {
	return map[string]any{
		"extract": extract,
	}
}

func extract(collection any, key any) any {
	k := fmt.Sprint(key)

	switch c := collection.(type) {
	case map[string]any:
		return c[k]
	case map[string]string:
		return c[k]
	case []any:
		if _, ok := convertToInt(key); ok {
			return extractFromSlice(len(c), key, func(i int) any { return c[i] })
		}

		return extractFromObjects(c, k)
	case []string:
		return extractFromSlice(len(c), key, func(i int) any { return c[i] })
	}

	return nil
}

func extractFromSlice(length int, key any, getter func(int) any) any {
	idx, ok := convertToInt(key)
	if !ok || idx < 0 || idx >= length {
		return nil
	}

	return getter(idx)
}

func extractFromObjects(items []any, key string) any {
	out := make([]any, 0, len(items))

	for _, item := range items {
		switch m := item.(type) {
		case map[string]any:
			if v, ok := m[key]; ok {
				out = append(out, v)
			}
		case map[string]string:
			if v, ok := m[key]; ok {
				out = append(out, v)
			}
		}
	}

	return out
}

func compareFuncs() map[string]any {
	cmpFn := func(a, b any) (int, bool) {
		va, okA := convertToFloat64(a)
		if !okA {
			return 0, false
		}

		vb, okB := convertToFloat64(b)
		if !okB {
			return 0, false
		}

		return cmp.Compare(va, vb), true
	}

	return map[string]any{
		"gt": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r > 0
			}

			return false
		},
		"lt": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r < 0
			}

			return false
		},
		"gte": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r >= 0
			}

			return false
		},
		"lte": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r <= 0
			}

			return false
		},
		"eq": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r == 0
			}

			return false
		},
	}
}
