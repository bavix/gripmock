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
		{name: "nil", value: nil, expected: true},
		{name: "empty string", value: "", expected: true},
		{name: "zero int", value: 0, expected: true},
		{name: "zero float", value: float64(0), expected: true},
		{name: "false bool", value: false, expected: true},
		{name: "non-empty string", value: "hello", expected: false},
		{name: "non-zero int", value: 42, expected: false},
		{name: "non-zero float", value: 3.14, expected: false},
		{name: "true bool", value: true, expected: false},
		{name: "map value", value: map[string]any{"a": 1}, expected: false},
		{name: "slice value", value: []any{1}, expected: false},
		{name: "enum unspecified string", value: "ENUM_UNSPECIFIED", expected: true},
		{name: "enum specified", value: "ACTIVE", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsDefaultValue(tt.value)
			require.Equal(t, tt.expected, result)
		})
	}
}
