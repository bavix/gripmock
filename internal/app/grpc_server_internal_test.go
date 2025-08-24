package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/types"
)

//nolint:funlen
func TestParseAsV4StreamStep(t *testing.T) {
	t.Parallel()

	mocker := &grpcMocker{}

	tests := []struct {
		name     string
		input    any
		expected *types.StreamStep
		hasError bool
	}{
		{
			name: "valid v4 StreamStep with send",
			input: map[string]any{
				"send": map[string]any{
					"message": "Hello",
					"id":      123,
				},
			},
			expected: &types.StreamStep{
				Send: map[string]any{
					"message": "Hello",
					"id":      123,
				},
			},
			hasError: false,
		},
		{
			name: "valid v4 StreamStep with delay",
			input: map[string]any{
				"delay": "100ms",
			},
			expected: &types.StreamStep{
				Delay: "100ms",
			},
			hasError: false,
		},
		{
			name: "valid v4 StreamStep with end",
			input: map[string]any{
				"end": map[string]any{
					"code":    "OK",
					"message": "Stream completed",
				},
			},
			expected: &types.StreamStep{
				End: &types.GrpcStatus{
					Code:    "OK",
					Message: "Stream completed",
				},
			},
			hasError: false,
		},
		{
			name: "valid v4 StreamStep with all fields",
			input: map[string]any{
				"send": map[string]any{
					"message": "Hello",
				},
				"delay": "50ms",
				"end": map[string]any{
					"code":    "OK",
					"message": "Done",
				},
			},
			expected: &types.StreamStep{
				Send: map[string]any{
					"message": "Hello",
				},
				Delay: "50ms",
				End: &types.GrpcStatus{
					Code:    "OK",
					Message: "Done",
				},
			},
			hasError: false,
		},
		{
			name:     "no v4 fields",
			input:    map[string]any{"legacy": "data"},
			expected: nil,
			hasError: true,
		},
		{
			name:     "not a map",
			input:    "string data",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := mocker.parseAsV4StreamStep(tt.input)

			if tt.hasError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
