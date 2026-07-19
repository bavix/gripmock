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
		"sum": sum,
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

func add(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok {
		return 0
	}

	sum := 0.0
	for _, v := range nums {
		sum += v
	}

	return sum
}

func subtract(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	result := nums[0]

	for _, v := range nums[1:] {
		result -= v
	}

	return result
}

func divide(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	result := nums[0]

	for _, v := range nums[1:] {
		if v != 0 {
			result /= v
		}
	}

	return result
}

func modulo(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) < 2 || nums[1] == 0 {
		return 0
	}

	return math.Mod(nums[0], nums[1])
}

func sum(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok {
		return 0
	}

	total := 0.0
	for _, v := range nums {
		total += v
	}

	return total
}

func product(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok {
		return 0
	}

	prod := 1.0
	for _, v := range nums {
		prod *= v
	}

	return prod
}

func average(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	total := 0.0
	for _, v := range nums {
		total += v
	}

	return total / float64(len(nums))
}

func minValue(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	minVal := nums[0]

	for _, v := range nums[1:] {
		minVal = minFloat(minVal, v)
	}

	return minVal
}

func maxValue(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	maxVal := nums[0]

	for _, v := range nums[1:] {
		maxVal = maxFloat(maxVal, v)
	}

	return maxVal
}

func convertAllToFloat64(values ...any) ([]float64, bool) {
	nums := make([]float64, 0, len(values))
	for _, v := range values {
		if f, ok := convertToFloat64(v); ok {
			nums = append(nums, f)
		} else {
			return nil, false
		}
	}

	return nums, true
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}
