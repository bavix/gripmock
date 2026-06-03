package proxycapture_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
)

func toStreamOutputWithDelays(responses []map[string]any, delays []time.Duration) []any {
	streamOutput := make([]any, 0, len(responses))
	for i, response := range responses {
		entry := make(map[string]any)
		if i < len(delays) && delays[i] > 0 {
			entry["delay"] = delays[i].String()
		}

		entry["data"] = response
		streamOutput = append(streamOutput, entry)
	}

	return streamOutput
}

func TestToStreamOutputWithDelays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		responses []map[string]any
		delays    []time.Duration
		wantLen   int
	}{
		{
			name:      "empty responses",
			responses: []map[string]any{},
			delays:    []time.Duration{},
			wantLen:   0,
		},
		{
			name:      "single response with delay",
			responses: []map[string]any{{"status": "ok"}},
			delays:    []time.Duration{100 * time.Millisecond},
			wantLen:   1,
		},
		{
			name:      "multiple responses with delays",
			responses: []map[string]any{{"id": 1}, {"id": 2}, {"id": 3}},
			delays:    []time.Duration{50 * time.Millisecond, 150 * time.Millisecond, 200 * time.Millisecond},
			wantLen:   3,
		},
		{
			name:      "more delays than responses - only first N used",
			responses: []map[string]any{{"id": 1}},
			delays:    []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			wantLen:   1,
		},
		{
			name:      "more responses than delays",
			responses: []map[string]any{{"id": 1}, {"id": 2}, {"id": 3}},
			delays:    []time.Duration{100 * time.Millisecond},
			wantLen:   3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := toStreamOutputWithDelays(tc.responses, tc.delays)
			require.Len(t, result, tc.wantLen)
		})
	}
}

func TestToStreamOutputWithDelays_DelayInEntry(t *testing.T) {
	t.Parallel()

	responses := []map[string]any{{"id": 1}, {"id": 2}}
	delays := []time.Duration{50 * time.Millisecond, 150 * time.Millisecond}

	result := toStreamOutputWithDelays(responses, delays)

	require.Len(t, result, 2)

	entry0, ok := result[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "50ms", entry0["delay"])
	require.Equal(t, responses[0], entry0["data"])

	entry1, ok := result[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "150ms", entry1["delay"])
	require.Equal(t, responses[1], entry1["data"])
}

func TestToStreamOutputWithDelays_NoDelayForIndex(t *testing.T) {
	t.Parallel()

	responses := []map[string]any{{"id": 1}, {"id": 2}}
	delays := []time.Duration{100 * time.Millisecond}

	result := toStreamOutputWithDelays(responses, delays)

	require.Len(t, result, 2)

	entry0, ok := result[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "100ms", entry0["delay"])
	require.Equal(t, responses[0], entry0["data"])

	entry1, ok := result[1].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, entry1, "delay")
	require.Equal(t, responses[1], entry1["data"])
}

func TestToStreamOutputWithDelays_ZeroDelay(t *testing.T) {
	t.Parallel()

	responses := []map[string]any{{"id": 1}}
	delays := []time.Duration{0}

	result := toStreamOutputWithDelays(responses, delays)

	require.Len(t, result, 1)

	entry0, ok := result[0].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, entry0, "delay")
}

func TestToStreamOutputWithDelays_VariousDurations(t *testing.T) {
	t.Parallel()

	responses := []map[string]any{
		{"id": 1},
		{"id": 2},
		{"id": 3},
		{"id": 4},
	}
	delays := []time.Duration{
		1 * time.Nanosecond,
		1 * time.Microsecond,
		1 * time.Millisecond,
		1 * time.Second,
	}

	result := toStreamOutputWithDelays(responses, delays)

	require.Len(t, result, 4)

	require.Equal(t, "1ns", result[0].(map[string]any)["delay"]) //nolint:forcetypeassert
	require.Equal(t, "1µs", result[1].(map[string]any)["delay"]) //nolint:forcetypeassert
	require.Equal(t, "1ms", result[2].(map[string]any)["delay"]) //nolint:forcetypeassert
	require.Equal(t, "1s", result[3].(map[string]any)["delay"])  //nolint:forcetypeassert
}

func TestBuildServerStreamStub(t *testing.T) {
	t.Parallel()

	stub := proxycapture.BuildServerStreamStub(
		"test.Service",
		"ServerStreamMethod",
		"session-srv",
		map[string]any{"request": "data"},
		map[string]any{"req-header": "value"},
		[]map[string]any{{"stream": 1}, {"stream": 2}, {"stream": 3}},
		map[string]string{"trailer": "value"},
		nil,
	)

	require.Equal(t, "test.Service", stub.Service)
	require.Equal(t, "ServerStreamMethod", stub.Method)
	require.Len(t, stub.Output.Stream, 3)
}
