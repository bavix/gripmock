package protoconv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsDefaultValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"unspecified enum", "LENGTH_UNIT_UNSPECIFIED", true},
		{"unspecified enum lowercase", "length_unit_unspecified", false},
		{"regular enum", "METERS", false},
		{"int zero", 0, true},
		{"int non-zero", 42, false},
		{"int negative", -1, false},
		{"int64 zero", int64(0), true},
		{"int64 non-zero", int64(42), false},
		{"float32 zero", float32(0), true},
		{"float32 non-zero", float32(3.14), false},
		{"float64 zero", float64(0), true},
		{"float64 non-zero", float64(2.71), false},
		{"bool false", false, true},
		{"bool true", true, false},
		{"empty slice", []any{}, true},
		{"non-empty slice", []any{1, 2, 3}, false},
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
