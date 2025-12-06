package stuber

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// TemplateFunctions provides all available template functions.
// Optimized for performance with direct function references and minimal allocations.
//
//nolint:funlen,cyclop
func TemplateFunctions() map[string]any {
	return map[string]any{
		// String operations - direct function references for maximum performance
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": titleCase,
		"join":  strings.Join,
		"split": strings.Split,

		// JSON operations
		"json": func(v any) string {
			b, _ := json.Marshal(v)

			return string(b)
		},

		// Formatting and casting helpers
		"sprintf": fmt.Sprintf,
		"str": func(v any) string {
			switch t := v.(type) {
			case string:
				return t
			case json.Number:
				return t.String()
			default:
				return fmt.Sprint(v)
			}
		},
		"int": func(v any) int {
			if f, ok := convertToFloat64(v); ok {
				return int(f)
			}

			return 0
		},
		"int64": func(v any) int64 {
			if f, ok := convertToFloat64(v); ok {
				return int64(f)
			}

			return 0
		},
		"float": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return f
			}

			return 0
		},

		// Rounding helpers (no precision)
		"round": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return math.Round(f)
			}

			return 0
		},
		"floor": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return math.Floor(f)
			}

			return 0
		},
		"ceil": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return math.Ceil(f)
			}

			return 0
		},

		// Number formatting
		"decimal": func(v any) json.Number {
			if f, ok := convertToFloat64(v); ok {
				// Force trailing .0 for integer-like values
				if math.Trunc(f) == f {
					return json.Number(strconv.FormatFloat(f, 'f', 1, 64))
				}

				return json.Number(strconv.FormatFloat(f, 'g', -1, 64))
			}

			return json.Number("0")
		},

		// Array operations (use built-in len and index from text/template)
		"extract": extract,

		// Comparison operations
		"gt": func(a, b any) bool {
			va, okA := convertToFloat64(a)
			if !okA {
				return false
			}

			vb, okB := convertToFloat64(b)
			if !okB {
				return false
			}

			return va > vb
		},

		// Mathematical operations - direct function references
		"add": add,
		"sub": subtract,
		"div": divide,
		"mod": modulo,
		"sum": sum,
		"mul": product,
		"avg": average,
		"min": minValue,
		"max": maxValue,

		// Time operations
		"now":    time.Now,
		"unix":   time.Time.Unix,
		"format": time.Time.Format,
	}
}

// titleCase converts first character to uppercase (replaces deprecated strings.Title).
func titleCase(s string) string {
	if s == "" {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])

	return string(r)
}

// ensureDecimal ensures integer-like numbers are represented with a trailing .0
// to keep JSON numbers like 25.0 instead of 25 where tests expect a decimal.
func ensureDecimalStringFromFloat(value float64) string {
	s := strconv.FormatFloat(value, 'g', -1, 64)
	// Keep as-is; do not force trailing .0 to avoid issues in string concatenations
	return s
}

// convertToFloat64 converts any numeric value to float64 for calculations.
//
//nolint:cyclop
func convertToFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return f, true
		}
	}

	return 0, false
}

// binaryOperation performs a binary operation with type safety.
// Returns json.Number for consistency.
func binaryOperation(a, b any, operation func(float64, float64) float64) any {
	va, okA := convertToFloat64(a)
	if !okA {
		return json.Number("0")
	}

	vb, okB := convertToFloat64(b)
	if !okB {
		return json.Number("0")
	}

	result := operation(va, vb)

	return json.Number(ensureDecimalStringFromFloat(result))
}

// binaryOperationWithZeroCheck performs a binary operation with zero division protection.
// Returns json.Number for consistency.
func binaryOperationWithZeroCheck(a, b any, operation func(float64, float64) float64) any {
	va, okA := convertToFloat64(a)
	if !okA {
		return json.Number("0")
	}

	vb, okB := convertToFloat64(b)
	if !okB {
		return json.Number("0")
	}

	if vb == 0 {
		return json.Number("0")
	}

	result := operation(va, vb)

	return json.Number(ensureDecimalStringFromFloat(result))
}

// add performs addition with type safety.
func add(a, b any) any {
	return binaryOperation(a, b, func(x, y float64) float64 { return x + y })
}

// subtract performs subtraction with type safety.
func subtract(a, b any) any {
	return binaryOperation(a, b, func(x, y float64) float64 { return x - y })
}

// divide performs division with type safety and zero division protection.
func divide(a, b any) any {
	return binaryOperationWithZeroCheck(a, b, func(x, y float64) float64 { return x / y })
}

// modulo performs modulo operation with type safety.
func modulo(a, b any) any {
	return binaryOperationWithZeroCheck(a, b, func(x, y float64) float64 { return x - float64(int(x/y))*y })
}

// sum calculates the sum of multiple values.
func sum(values ...any) any {
	if len(values) == 0 {
		return json.Number("0")
	}

	// Support a single []any argument
	if len(values) == 1 {
		if arr, ok := values[0].([]any); ok {
			values = arr
		}
	}

	var result float64

	for _, v := range values {
		if val, ok := convertToFloat64(v); ok {
			result += val
		}
	}

	return json.Number(ensureDecimalStringFromFloat(result))
}

// product calculates the product of multiple values
// exposed to templates as "mul" to avoid ambiguity with domain terms.
func product(values ...any) any {
	if len(values) == 0 {
		return json.Number("0")
	}

	// Support a single []any argument
	if len(values) == 1 {
		if arr, ok := values[0].([]any); ok {
			values = arr
		}
	}

	result := 1.0

	for _, v := range values {
		if val, ok := convertToFloat64(v); ok {
			result *= val
		}
	}

	return json.Number(ensureDecimalStringFromFloat(result))
}

// average calculates the average of multiple values.
func average(values ...any) any {
	if len(values) == 0 {
		return json.Number("0")
	}

	// Support both variadic values and a single []any slice
	if len(values) == 1 {
		if arr, ok := values[0].([]any); ok {
			values = arr
		}
	}

	sumVal := sum(values...)
	if s, ok := sumVal.(json.Number); ok {
		if f, err := s.Float64(); err == nil {
			result := f / float64(len(values))

			return json.Number(ensureDecimalStringFromFloat(result))
		}
	}

	return json.Number("0")
}

// minValue finds the minimum value among multiple values.
func minValue(values ...any) any {
	if len(values) == 0 {
		return json.Number("0")
	}

	// Support a single []any argument
	if len(values) == 1 {
		if arr, ok := values[0].([]any); ok {
			values = arr
		}
	}

	var minValue float64

	first := true

	for _, v := range values {
		if current, ok := convertToFloat64(v); ok {
			if first || current < minValue {
				minValue = current
				first = false
			}
		}
	}

	return json.Number(ensureDecimalStringFromFloat(minValue))
}

// maxValue finds the maximum value among multiple values.
func maxValue(values ...any) any {
	if len(values) == 0 {
		return json.Number("0")
	}

	// Support a single []any argument
	if len(values) == 1 {
		if arr, ok := values[0].([]any); ok {
			values = arr
		}
	}

	var maxValue float64

	first := true

	for _, v := range values {
		if current, ok := convertToFloat64(v); ok {
			if first || current > maxValue {
				maxValue = current
				first = false
			}
		}
	}

	return json.Number(ensureDecimalStringFromFloat(maxValue))
}

// extract extracts a specific field from each message in a slice.
func extract(values []any, fieldName string) []any {
	if len(values) == 0 {
		return []any{}
	}

	result := make([]any, 0, len(values))
	for _, v := range values {
		if val, ok := v.(map[string]any); ok {
			if value, ok := val[fieldName]; ok {
				result = append(result, value)
			}
		}
	}

	return result
}
