package plugins

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

func numberFuncs() map[string]any {
	return map[string]any{
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
		"decimal": func(v any) json.Number {
			if f, ok := convertToFloat64(v); ok {
				if math.Trunc(f) == f {
					return json.Number(strconv.FormatFloat(f, 'f', 1, 64))
				}

				return json.Number(strconv.FormatFloat(f, 'g', -1, 64))
			}

			return json.Number("0")
		},
	}
}

func mathFuncs() map[string]any {
	return map[string]any{
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
		"add": add,
		"sub": subtract,
		"div": divide,
		"mod": modulo,
		"sum": add,
		"mul": product,
		"avg": average,
		"min": minValue,
		"max": maxValue,
	}
}

func convertToFloat64(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case json.Number:
		f, err := value.Float64()
		if err == nil {
			return f, true
		}
	case string:
		f, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return f, true
		}
	default:
		f, err := strconv.ParseFloat(fmt.Sprint(value), 64)
		if err == nil {
			return f, true
		}
	}

	return 0, false
}

func convertToInt(v any) (int, bool) {
	switch value := v.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case float32:
		return int(value), true
	case json.Number:
		if i, err := value.Int64(); err == nil {
			return int(i), true
		}
	case string:
		return parseIntString(value)
	default:
		return parseIntString(fmt.Sprint(value))
	}

	return 0, false
}

func parseIntString(s string) (int, bool) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}

	return i, true
}

func foldFloats(nums []float64, seed float64, seedFromFirst bool, op func(a, b float64) float64) (float64, int) {
	acc, count := seed, 0

	for _, f := range nums {
		if seedFromFirst && count == 0 {
			acc = f
		} else {
			acc = op(acc, f)
		}

		count++
	}

	return acc, count
}

func flattenNumbers(values []any) []any {
	if len(values) == 1 {
		if nested, ok := values[0].([]any); ok {
			return nested
		}
	}

	return values
}

func foldNumbers(values []any, seed float64, seedFromFirst bool, op func(a, b float64) float64) (float64, int, bool) {
	if len(values) == 1 {
		if nums, ok := values[0].([]float64); ok {
			acc, count := foldFloats(nums, seed, seedFromFirst, op)

			return acc, count, true
		}
	}

	acc, count := seed, 0

	for _, raw := range flattenNumbers(values) {
		f, ok := convertToFloat64(raw)
		if !ok {
			return 0, 0, false
		}

		if seedFromFirst && count == 0 {
			acc = f
		} else {
			acc = op(acc, f)
		}

		count++
	}

	return acc, count, true
}

func fold(values []any, seed float64, seedFromFirst bool, op func(a, b float64) float64) float64 {
	acc, _, ok := foldNumbers(values, seed, seedFromFirst, op)
	if !ok {
		return 0
	}

	return acc
}

func add(values ...any) float64 {
	return fold(values, 0, false, func(a, b float64) float64 { return a + b })
}

func subtract(values ...any) float64 {
	return fold(values, 0, true, func(a, b float64) float64 { return a - b })
}

func divide(values ...any) float64 {
	return fold(values, 0, true, func(a, b float64) float64 {
		if b == 0 {
			return a
		}

		return a / b
	})
}

func modulo(values ...any) float64 {
	if len(values) < 2 { //nolint:mnd
		return 0
	}

	first, okFirst := convertToFloat64(values[0])
	second, okSecond := convertToFloat64(values[1])

	if !okFirst || !okSecond || second == 0 {
		return 0
	}

	return math.Mod(first, second)
}

func product(values ...any) float64 {
	return fold(values, 1, false, func(a, b float64) float64 { return a * b })
}

func average(values ...any) float64 {
	total, count, ok := foldNumbers(values, 0, false, func(a, b float64) float64 { return a + b })
	if !ok || count == 0 {
		return 0
	}

	return total / float64(count)
}

func minValue(values ...any) float64 {
	return fold(values, 0, true, math.Min)
}

func maxValue(values ...any) float64 {
	return fold(values, 0, true, math.Max)
}
