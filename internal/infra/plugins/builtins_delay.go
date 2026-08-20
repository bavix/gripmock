package plugins

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

func delayFuncs() map[string]any {
	return map[string]any{
		"regressive": regressiveDelay,
		"backoff":    backoffDelay,
		"jitter":     jitterDelay,
	}
}

func regressiveDelay(attempt, start, step any) string {
	n, okAttempt := convertToFloat64(attempt)
	from, okStart := toDuration(start)
	by, okStep := toDuration(step)

	if !okAttempt || !okStart || !okStep {
		return "0s"
	}

	return max(0, from-time.Duration(math.Max(0, n-1))*by).String()
}

func backoffDelay(attempt, base any, limit ...any) string {
	n, okAttempt := convertToFloat64(attempt)
	first, okBase := toDuration(base)

	if !okAttempt || !okBase {
		return "0s"
	}

	value := time.Duration(float64(first) * math.Pow(2, math.Max(0, n-1))) //nolint:mnd

	if len(limit) > 0 {
		capped, ok := toDuration(limit[0])
		if !ok {
			return "0s"
		}

		value = min(value, capped)
	}

	return value.String()
}

func jitterDelay(low, high any) string {
	from, okLow := toDuration(low)
	to, okHigh := toDuration(high)

	if !okLow || !okHigh || to < from {
		return "0s"
	}

	if to == from {
		return from.String()
	}

	span, err := rand.Int(rand.Reader, big.NewInt(int64(to-from)))
	if err != nil {
		return from.String()
	}

	return (from + time.Duration(span.Int64())).String()
}

func toDuration(v any) (time.Duration, bool) {
	switch value := v.(type) {
	case time.Duration:
		return value, true
	case string:
		parsed, err := time.ParseDuration(value)

		return parsed, err == nil
	default:
		if f, ok := convertToFloat64(v); ok {
			return time.Duration(f) * time.Millisecond, true
		}

		return 0, false
	}
}
