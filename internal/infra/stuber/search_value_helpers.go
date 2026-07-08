package stuber

import (
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
