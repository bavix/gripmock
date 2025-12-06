package types_test

import (
	"encoding/json"
	"testing"
	"time"

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
	}

	for _, testCase := range tests {
		// Go 1.22+ runs subtests in parallel safely without copying loop var.
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var duration types.Duration

			err := json.Unmarshal([]byte(testCase.input), &duration)

			if (err != nil) != testCase.wantErr {
				t.Errorf("Duration.UnmarshalJSON() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if !testCase.wantErr && time.Duration(duration) != testCase.want {
				t.Errorf("Duration.UnmarshalJSON() = %v, want %v", time.Duration(duration), testCase.want)
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	duration := types.Duration(100 * time.Millisecond)
	expected := `"100ms"`

	got, err := json.Marshal(duration)
	if err != nil {
		t.Errorf("Duration.MarshalJSON() error = %v", err)

		return
	}

	if string(got) != expected {
		t.Errorf("Duration.MarshalJSON() = %v, want %v", string(got), expected)
	}
}
