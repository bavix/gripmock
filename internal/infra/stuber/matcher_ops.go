package stuber

import (
	"errors"
	"path"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

var errFail = errors.New("glob match failed")

// contains checks if the expected map is a subset of the actual value.
//
// It returns true if the expected map is a subset of the actual value,
// otherwise false.
func contains(expected map[string]any, actual any) bool {
	if len(expected) == 0 {
		return true
	}

	return deeply.ContainsIgnoreArrayOrder(expected, actual)
}

// matches checks if the expected map matches the actual value using regular expressions.
//
// It returns true if the expected map matches the actual value using regular expressions,
// otherwise false.
func matches(expected map[string]any, actual any) bool {
	if len(expected) == 0 {
		return true
	}

	return deeply.MatchesIgnoreArrayOrder(expected, actual)
}

// globMatch checks if the expected map matches the actual value using glob patterns.
//
// It returns true if all glob patterns match, otherwise false.
// Supports nested map traversal for matching at any depth.
func globMatch(expected map[string]any, actual any) bool {
	if len(expected) == 0 {
		return true
	}

	actualMap, ok := actual.(map[string]any)
	if !ok {
		return false
	}

	for key, pattern := range expected {
		actualValue, exists := actualMap[key]
		if !exists {
			return false
		}

		if err := matchGlobValue(pattern, actualValue); err != nil {
			return false
		}
	}

	return true
}

// matchGlobValue matches a pattern against a value, supporting nested maps.
func matchGlobValue(pattern, actual any) error {
	patternStr, isStringPattern := pattern.(string)
	actualStr, isStringActual := actual.(string)

	if isStringPattern && isStringActual {
		matched, err := path.Match(patternStr, actualStr)
		if err != nil || !matched {
			return errFail
		}

		return nil
	}

	patternMap, isPatternMap := pattern.(map[string]any)
	actualMap, isActualMap := actual.(map[string]any)

	if isPatternMap && isActualMap {
		for key, pat := range patternMap {
			act, exists := actualMap[key]
			if !exists {
				return errFail
			}

			if err := matchGlobValue(pat, act); err != nil {
				return err
			}
		}

		return nil
	}

	return errFail
}

// rankGlob calculates rank for glob pattern matching.
func rankGlob(expected map[string]any, actual any) float64 {
	if len(expected) == 0 {
		return 0
	}

	return rankGlobValue(expected, actual)
}

func rankGlobValue(pattern, actual any) float64 {
	patternStr, isStringPattern := pattern.(string)
	actualStr, isStringActual := actual.(string)

	if isStringPattern && isStringActual {
		matched, err := path.Match(patternStr, actualStr)
		if err == nil && matched {
			return 1.0
		}

		return 0
	}

	patternMap, isPatternMap := pattern.(map[string]any)
	actualMap, isActualMap := actual.(map[string]any)

	if isPatternMap && isActualMap {
		var rank float64

		for key, pat := range patternMap {
			act, exists := actualMap[key]
			if !exists {
				continue
			}

			rank += rankGlobValue(pat, act)
		}

		return rank
	}

	return 0
}
