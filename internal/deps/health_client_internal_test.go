package deps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePingAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ipv4 wildcard",
			input:    "0.0.0.0:4770",
			expected: "127.0.0.1:4770",
		},
		{
			name:     "ipv6 wildcard",
			input:    "[::]:4770",
			expected: "[::1]:4770",
		},
		{
			name:     "localhost unchanged",
			input:    "127.0.0.1:4770",
			expected: "127.0.0.1:4770",
		},
		{
			name:     "hostname unchanged",
			input:    "example.local:4770",
			expected: "example.local:4770",
		},
		{
			name:     "invalid address unchanged",
			input:    "invalid",
			expected: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			input := tt.input

			// Act
			actual := normalizePingAddress(input)

			// Assert
			require.Equal(t, tt.expected, actual)
		})
	}
}
