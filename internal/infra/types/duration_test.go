package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestDuration_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "string with ms",
			input:   `"100ms"`,
			want:    100 * time.Millisecond,
			wantErr: false,
		},
		{
			name:    "string with seconds",
			input:   `"2s"`,
			want:    2 * time.Second,
			wantErr: false,
		},
		{
			name:    "invalid duration string",
			input:   `"invalid"`,
			want:    0,
			wantErr: true,
		},
		{
			name:    "numeric nanoseconds",
			input:   "1000000000",
			want:    time.Second,
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		// Go 1.22+ runs subtests in parallel safely without copying loop var.
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var duration types.Duration

			// Act
			err := json.Unmarshal([]byte(testCase.input), &duration)

			// Assert
			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.want, time.Duration(duration))
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	duration := types.Duration(100 * time.Millisecond)
	expected := `"100ms"`

	got, err := json.Marshal(duration)
	require.NoError(t, err)
	require.Equal(t, expected, string(got))
}
