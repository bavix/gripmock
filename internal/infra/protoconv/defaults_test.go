package protoconv //nolint:testpackage // Tests internal implementation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type filterDefaultValuesCase struct {
	name     string
	input    map[string]any
	expected map[string]any
}

type convertMapCase struct {
	name     string
	input    map[string]any
	handling DefaultValueHandling
	expected map[string]any
}

func filterDefaultValuesCases() []filterDefaultValuesCase {
	return []filterDefaultValuesCase{
		{name: "nil input", input: nil, expected: nil},
		{name: "empty input", input: map[string]any{}, expected: map[string]any{}},
		{
			name:     "all defaults",
			input:    map[string]any{"field1": "", "field2": 0, "field3": false, "field4": "ENUM_UNSPECIFIED"},
			expected: map[string]any{},
		},
		{
			name:     "no defaults",
			input:    map[string]any{"field1": "value", "field2": 42, "field3": true},
			expected: map[string]any{"field1": "value", "field2": 42, "field3": true},
		},
		{
			name: "mixed values",
			input: map[string]any{
				"value":     1000,
				"to_unit":   "METERS",
				"from_unit": "LENGTH_UNIT_UNSPECIFIED",
				"empty":     "",
				"zero":      0,
				"non_zero":  5,
			},
			expected: map[string]any{"value": 1000, "to_unit": "METERS", "non_zero": 5},
		},
		{
			name:     "nested structures preserved",
			input:    map[string]any{"nested": map[string]any{"key": "value"}, "empty": "", "list": []any{1, 2, 3}},
			expected: map[string]any{"nested": map[string]any{"key": "value"}, "list": []any{1, 2, 3}},
		},
	}
}

func convertMapCases() []convertMapCase {
	return []convertMapCase{
		{name: "nil input with IncludeDefaults", input: nil, handling: IncludeDefaults, expected: nil},
		{name: "nil input with ExcludeDefaults", input: nil, handling: ExcludeDefaults, expected: nil},
		{
			name:     "IncludeDefaults preserves all fields",
			input:    map[string]any{"value": 1000, "to_unit": "METERS", "from_unit": "LENGTH_UNIT_UNSPECIFIED", "empty": "", "zero": 0},
			handling: IncludeDefaults,
			expected: map[string]any{"value": 1000, "to_unit": "METERS", "from_unit": "LENGTH_UNIT_UNSPECIFIED", "empty": "", "zero": 0},
		},
		{
			name:     "ExcludeDefaults filters default values",
			input:    map[string]any{"value": 1000, "to_unit": "METERS", "from_unit": "LENGTH_UNIT_UNSPECIFIED", "empty": "", "zero": 0},
			handling: ExcludeDefaults,
			expected: map[string]any{"value": 1000, "to_unit": "METERS"},
		},
		{name: "empty map with IncludeDefaults", input: map[string]any{}, handling: IncludeDefaults, expected: map[string]any{}},
		{name: "empty map with ExcludeDefaults", input: map[string]any{}, handling: ExcludeDefaults, expected: map[string]any{}},
	}
}

func TestIsDefaultValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		// Nil values
		{"nil", nil, true},

		// String values
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"unspecified enum", "LENGTH_UNIT_UNSPECIFIED", true},
		{"unspecified enum lowercase", "length_unit_unspecified", false},
		{"regular enum", "METERS", false},

		// Integer values
		{"int zero", 0, true},
		{"int non-zero", 42, false},
		{"int negative", -1, false},
		{"int64 zero", int64(0), true},
		{"int64 non-zero", int64(42), false},

		// Float values
		{"float32 zero", float32(0), true},
		{"float32 non-zero", float32(3.14), false},
		{"float64 zero", float64(0), true},
		{"float64 non-zero", float64(2.71), false},

		// Boolean values
		{"bool false", false, true},
		{"bool true", true, false},

		// Slice values
		{"empty slice", []any{}, true},
		{"non-empty slice", []any{1, 2, 3}, false},

		// Map values
		{"empty map", map[string]any{}, true},
		{"non-empty map", map[string]any{"key": "value"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsDefaultValue(tt.value)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterDefaultValues(t *testing.T) {
	t.Parallel()

	for _, tt := range filterDefaultValuesCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := FilterDefaultValues(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterDefaultValuesDoesNotModifyOriginal(t *testing.T) {
	t.Parallel()

	original := map[string]any{
		"keep": "value",
		"drop": "",
		"also": 0,
	}

	// Make a copy to compare
	expectedOriginal := map[string]any{
		"keep": "value",
		"drop": "",
		"also": 0,
	}

	_ = FilterDefaultValues(original)

	require.Equal(t, expectedOriginal, original, "original map should not be modified")
}

func TestConvertMap(t *testing.T) {
	t.Parallel()

	for _, tt := range convertMapCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ConvertMap(tt.input, tt.handling)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertMapDoesNotModifyOriginal(t *testing.T) {
	t.Parallel()

	original := map[string]any{
		"keep": "value",
		"drop": "",
		"also": 0,
	}

	expectedOriginal := map[string]any{
		"keep": "value",
		"drop": "",
		"also": 0,
	}

	_ = ConvertMap(original, ExcludeDefaults)

	require.Equal(t, expectedOriginal, original, "original map should not be modified")
}

func TestConvertMapIncludeDefaultsReturnsCopy(t *testing.T) {
	t.Parallel()

	original := map[string]any{
		"field1": "value1",
		"field2": 42,
	}

	result := ConvertMap(original, IncludeDefaults)

	// Verify content is the same
	require.Equal(t, original, result)

	// Verify it's a different map (modifying result doesn't affect original)
	result["new_field"] = "new_value"

	require.NotContains(t, original, "new_field")
}
