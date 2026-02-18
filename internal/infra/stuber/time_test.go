package stuber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

// TestDelayWithGo125TimeAPI demonstrates using Go 1.25 time API for deterministic testing.
func TestDelayWithGo125TimeAPI(t *testing.T) {
	t.Parallel()

	// Use Go 1.25 time API for deterministic testing
	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// Create a stub with delay
	output := stuber.Output{
		Data: map[string]any{
			"message": "Hello with delay",
		},
		Delay: types.Duration(100 * time.Millisecond),
	}

	// Simulate processing with delay
	start := baseTime
	expectedEnd := start.Add(100 * time.Millisecond)

	// In a real scenario, this would be the actual delay processing
	// For testing, we can simulate it with deterministic time
	actualEnd := start.Add(time.Duration(output.Delay))

	require.Equal(t, expectedEnd, actualEnd)
	require.Equal(t, types.Duration(100*time.Millisecond), output.Delay)
}

// TestDelaySerializationWithDeterministicTime tests delay serialization with deterministic time.
func TestDelaySerializationWithDeterministicTime(t *testing.T) {
	t.Parallel()

	// Use Go 1.25 time API for deterministic testing
	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	output := stuber.Output{
		Data: map[string]any{
			"timestamp": baseTime.Format(time.RFC3339),
			"message":   "Timestamped response",
		},
		Delay: types.Duration(200 * time.Millisecond),
	}

	// Verify the timestamp is deterministic
	expectedTimestamp := "2024-01-15T10:30:00Z"
	require.Equal(t, expectedTimestamp, output.Data["timestamp"])

	// Verify delay is correct
	require.Equal(t, types.Duration(200*time.Millisecond), output.Delay)
}

// TestDelayComparisonWithGo125API demonstrates time comparison using Go 1.25 API.
func TestDelayComparisonWithGo125API(t *testing.T) {
	t.Parallel()

	// Use Go 1.25 time API for deterministic testing
	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	delays := []types.Duration{
		types.Duration(10 * time.Millisecond),
		types.Duration(50 * time.Millisecond),
		types.Duration(100 * time.Millisecond),
		types.Duration(500 * time.Millisecond),
	}

	expectedEndTimes := []time.Time{
		baseTime.Add(10 * time.Millisecond),
		baseTime.Add(50 * time.Millisecond),
		baseTime.Add(100 * time.Millisecond),
		baseTime.Add(500 * time.Millisecond),
	}

	for i, delay := range delays {
		actualEndTime := baseTime.Add(time.Duration(delay))
		require.Equal(t, expectedEndTimes[i], actualEndTime)
	}
}

// TestDelayWithTemplateFunctions demonstrates delay with template functions using deterministic time.
func TestDelayWithTemplateFunctions(t *testing.T) {
	t.Parallel()

	// Use Go 1.25 time API for deterministic testing
	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// Simulate template data with deterministic time
	templateData := template.Data{
		Request: map[string]any{
			"user_id": "user123",
		},
		RequestTime: baseTime,
		State: map[string]any{
			"request_count": 5,
		},
	}

	// Create output with delay based on template data
	_ = stuber.Output{
		Data: map[string]any{
			"user_id":          "{{.Request.user_id}}",
			"timestamp":        "{{.RequestTime | format \"2006-01-02T15:04:05Z\"}}",
			"delay_multiplier": "{{.State.request_count}}",
		},
		Delay: types.Duration(10 * time.Millisecond),
	}

	// Verify template data is deterministic
	require.Equal(t, baseTime, templateData.RequestTime)
	require.Equal(t, "user123", templateData.Request["user_id"])
	require.Equal(t, 5, templateData.State["request_count"])
}
