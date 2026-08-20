package stuber_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestOutputDelayJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		payload   string
		encoded   string
		delay     types.Delay
		static    time.Duration
		wantError bool
	}{
		{name: "duration string", payload: `{"delay":"100ms"}`, encoded: `"delay":"100ms"`, delay: "100ms", static: 100 * time.Millisecond},
		{name: "nanoseconds", payload: `{"delay":100}`, encoded: `"delay":"100ns"`, delay: "100ns", static: 100},
		{name: "absent", payload: `{}`},
		{
			name:    "template",
			payload: `{"delay":"{{ duration .AttemptNumber }}"}`,
			encoded: `"delay":"{{ duration .AttemptNumber }}"`,
			delay:   "{{ duration .AttemptNumber }}",
		},
		{name: "invalid", payload: `{"delay":"banana"}`, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output stuber.Output

			err := json.Unmarshal([]byte(tt.payload), &output)
			if tt.wantError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.delay, output.Delay)
			require.Equal(t, tt.static, time.Duration(output.Delay.Static()))

			encoded, err := json.Marshal(output)
			require.NoError(t, err)

			if tt.encoded == "" {
				require.NotContains(t, string(encoded), "delay")
			} else {
				require.Contains(t, string(encoded), tt.encoded)
			}
		})
	}
}

func TestExtractGripMockDelayTemplate(t *testing.T) {
	t.Parallel()

	element := stuber.ExtractGripMock(map[string]any{
		stuber.GripMockKey: map[string]any{"delay": "{{ duration 50 }}"},
	})

	require.True(t, element.HasDelay)
	require.Equal(t, types.Delay("{{ duration 50 }}"), element.Delay)
	require.Zero(t, element.Delay.Static())

	static := stuber.ExtractGripMock(map[string]any{
		stuber.GripMockKey: map[string]any{"delay": "50ms"},
	})

	require.True(t, static.HasDelay)
	require.Equal(t, 50*time.Millisecond, time.Duration(static.Delay.Static()))
}

func TestMatchNumberIncrementsPerSession(t *testing.T) {
	t.Parallel()

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(&stuber.Stub{
		ID:      uuid.New(),
		Service: "Greeter",
		Method:  "SayHello",
		Input:   stuber.InputData{Equals: map[string]any{"name": "gripmock"}},
	})

	query := stuber.Query{
		Service: "Greeter",
		Method:  "SayHello",
		Input:   []map[string]any{{"name": "gripmock"}},
	}

	for i := 1; i <= 3; i++ {
		result, err := budgerigar.FindByQuery(query)
		require.NoError(t, err)
		require.Equal(t, i, result.MatchNumber())
	}

	query.Session = "other"

	result, err := budgerigar.FindByQuery(query)
	require.NoError(t, err)
	require.Equal(t, 1, result.MatchNumber())
}
