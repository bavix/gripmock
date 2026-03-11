package stuber

import (
	"fmt"
	"strings"
	"unicode"
)

// toCamelCase converts snake_case to camelCase.
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}

	result := parts[0]

	var builder strings.Builder

	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			builder.WriteString(strings.ToUpper(parts[i][:1]) + parts[i][1:])
		}
	}

	result += builder.String()

	return result
}

// toSnakeCase converts camelCase to snake_case.
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteByte('_')
		}

		result.WriteRune(unicode.ToLower(r))
	}

	return result.String()
}

// deepEqual performs deep equality check with better implementation.
func deepEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	switch a.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return a == b
	}

	if eq, handled := deepEqualMap(a, b); handled {
		return eq
	}

	if eq, handled := deepEqualSlice(a, b); handled {
		return eq
	}

	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func deepEqualMap(a, b any) (bool, bool) {
	aMap, aOk := a.(map[string]any)

	bMap, bOk := b.(map[string]any)
	if !aOk || !bOk {
		return false, false
	}

	if len(aMap) != len(bMap) {
		return false, true
	}

	for k, v := range aMap {
		if bv, exists := bMap[k]; !exists || !deepEqual(v, bv) {
			return false, true
		}
	}

	return true, true
}

func deepEqualSlice(a, b any) (bool, bool) {
	aSlice, aOk := a.([]any)

	bSlice, bOk := b.([]any)
	if !aOk || !bOk {
		return false, false
	}

	if len(aSlice) != len(bSlice) {
		return false, true
	}

	for i, v := range aSlice {
		if !deepEqual(v, bSlice[i]) {
			return false, true
		}
	}

	return true, true
}
