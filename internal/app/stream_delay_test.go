package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestExtractStreamDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		item         any
		wantDelay    types.Duration
		wantHasDelay bool
		wantErr      bool
	}{
		{
			name:         "item is not a map",
			item:         "not a map",
			wantDelay:    0,
			wantHasDelay: false,
			wantErr:      false,
		},
		{
			name:         "map without delay key",
			item:         map[string]any{"data": map[string]any{"key": "value"}},
			wantDelay:    0,
			wantHasDelay: false,
			wantErr:      false,
		},
		{
			name:         "map with delay key but nil value - treated as no delay",
			item:         map[string]any{"delay": nil},
			wantDelay:    0,
			wantHasDelay: false,
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotDelay, gotHasDelay, gotErr := extractStreamDelay(tc.item)
			require.Equal(t, tc.wantDelay, gotDelay)
			require.Equal(t, tc.wantHasDelay, gotHasDelay)

			if tc.wantErr {
				require.Error(t, gotErr)
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}

func TestExtractStreamDelayFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		item         any
		wantDelay    types.Duration
		wantHasDelay bool
		wantErr      bool
	}{
		{
			name:         "delay as string milliseconds",
			item:         map[string]any{"delay": "50ms"},
			wantDelay:    50 * 1e6,
			wantHasDelay: true,
			wantErr:      false,
		},
		{
			name:         "delay as string seconds",
			item:         map[string]any{"delay": "2s"},
			wantDelay:    2e9,
			wantHasDelay: true,
			wantErr:      false,
		},
		{
			name:         "delay as invalid string",
			item:         map[string]any{"delay": "not a duration"},
			wantDelay:    0,
			wantHasDelay: true,
			wantErr:      true,
		},
		{
			name:         "delay as float64 nanoseconds",
			item:         map[string]any{"delay": float64(100000000)},
			wantDelay:    100 * 1e6,
			wantHasDelay: true,
			wantErr:      false,
		},
		{
			name:         "delay as integer nanoseconds",
			item:         map[string]any{"delay": int64(250000000)},
			wantDelay:    250 * 1e6,
			wantHasDelay: true,
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotDelay, gotHasDelay, gotErr := extractStreamDelay(tc.item)
			require.Equal(t, tc.wantDelay, gotDelay)
			require.Equal(t, tc.wantHasDelay, gotHasDelay)

			if tc.wantErr {
				require.Error(t, gotErr)
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}

func TestExtractStreamData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		item      any
		wantData  map[string]any
		wantValid bool
	}{
		{
			name:      "item is not a map",
			item:      "not a map",
			wantData:  nil,
			wantValid: false,
		},
		{
			name:      "plain map without data key",
			item:      map[string]any{"key": "value"},
			wantData:  map[string]any{"key": "value"},
			wantValid: true,
		},
		{
			name:      "map with explicit data key",
			item:      map[string]any{"data": map[string]any{"key": "value"}},
			wantData:  map[string]any{"key": "value"},
			wantValid: true,
		},
		{
			name:      "map with data and delay",
			item:      map[string]any{"data": map[string]any{"status": "SERVING"}, "delay": "50ms"},
			wantData:  map[string]any{"status": "SERVING"},
			wantValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotData, gotValid := extractStreamData(tc.item)
			require.Equal(t, tc.wantData, gotData)
			require.Equal(t, tc.wantValid, gotValid)
		})
	}
}
