package stuber

import (
	"encoding/json"
	"testing"
)

// FuzzEquals tests the equals function against random JSON inputs
// to ensure it doesn't panic.
func FuzzEquals(f *testing.F) {
	f.Add([]byte(`{"a": 1}`), []byte(`{"a": 1}`), false)
	f.Add([]byte(`{"a": 1}`), []byte(`{"a": 2}`), true)
	f.Add([]byte(`{"a": [1, 2]}`), []byte(`{"a": [2, 1]}`), true)
	f.Add([]byte(`{"a": {"b": "c"}}`), []byte(`{"a": {"b": "c"}}`), false)
	f.Add([]byte(`bad json`), []byte(`bad json`), false)

	f.Fuzz(func(t *testing.T, expectedBytes, actualBytes []byte, orderIgnore bool) {
		var expected map[string]any
		if err := json.Unmarshal(expectedBytes, &expected); err != nil {
			return // Skip invalid JSON
		}

		var actual any
		if err := json.Unmarshal(actualBytes, &actual); err != nil {
			return // Skip invalid JSON
		}

		// The goal of fuzzing here is to ensure no panic occurs
		_ = equals(expected, actual, orderIgnore)
	})
}

// FuzzContains tests the contains function against random JSON inputs
// to ensure it doesn't panic.
func FuzzContains(f *testing.F) {
	f.Add([]byte(`{"a": 1}`), []byte(`{"a": 1, "b": 2}`), false)
	f.Add([]byte(`{"a": 1}`), []byte(`{"a": 2}`), true)
	f.Add([]byte(`bad json`), []byte(`bad json`), false)

	f.Fuzz(func(t *testing.T, expectedBytes, actualBytes []byte, orderIgnore bool) {
		var expected map[string]any
		if err := json.Unmarshal(expectedBytes, &expected); err != nil {
			return // Skip invalid JSON
		}

		var actual any
		if err := json.Unmarshal(actualBytes, &actual); err != nil {
			return // Skip invalid JSON
		}

		_ = contains(expected, actual, orderIgnore)
	})
}

// FuzzMatches tests the matches function against random JSON inputs
// to ensure it doesn't panic on malformed regex or deep structures.
func FuzzMatches(f *testing.F) {
	f.Add([]byte(`{"a": "^match.*"}`), []byte(`{"a": "matches this"}`), false)
	f.Add([]byte(`{"a": "["}`), []byte(`{"a": "b"}`), true) // bad regex
	f.Add([]byte(`bad json`), []byte(`bad json`), false)

	f.Fuzz(func(t *testing.T, expectedBytes, actualBytes []byte, orderIgnore bool) {
		var expected map[string]any
		if err := json.Unmarshal(expectedBytes, &expected); err != nil {
			return // Skip invalid JSON
		}

		var actual any
		if err := json.Unmarshal(actualBytes, &actual); err != nil {
			return // Skip invalid JSON
		}

		_ = matches(expected, actual, orderIgnore)
	})
}

// FuzzGlob tests the glob function against random JSON inputs
// to ensure it doesn't panic on malformed patterns or deep structures.
func FuzzGlob(f *testing.F) {
	f.Add([]byte(`{"a": "*.txt"}`), []byte(`{"a": "file.txt"}`))
	f.Add([]byte(`{"a": "["}`), []byte(`{"a": "b"}`)) // bad glob pattern
	f.Add([]byte(`bad json`), []byte(`bad json`))
	f.Add([]byte(`{"a": {"b": "*.txt"}}`), []byte(`{"a": {"b": "file.txt"}}`))

	f.Fuzz(func(t *testing.T, expectedBytes, actualBytes []byte) {
		var expected map[string]any
		if err := json.Unmarshal(expectedBytes, &expected); err != nil {
			return // Skip invalid JSON
		}

		var actual any
		if err := json.Unmarshal(actualBytes, &actual); err != nil {
			return // Skip invalid JSON
		}

		_ = globMatch(expected, actual)
	})
}

// FuzzRankMatch tests the deeply.RankMatch function which is used heavily
// by matcher.go.
func FuzzRankMatch(f *testing.F) {
	f.Add([]byte(`{"a": 1}`), []byte(`{"a": 1}`))
	f.Add([]byte(`bad json`), []byte(`bad json`))

	f.Fuzz(func(t *testing.T, expectedBytes, actualBytes []byte) {
		var expected map[string]any
		if err := json.Unmarshal(expectedBytes, &expected); err != nil {
			return // Skip invalid JSON
		}

		var actual any
		if err := json.Unmarshal(actualBytes, &actual); err != nil {
			return // Skip invalid JSON
		}

		//nolint:forcetypeassert
		_ = rankInput(actual.(map[string]any), InputData{Equals: expected})
	})
}
