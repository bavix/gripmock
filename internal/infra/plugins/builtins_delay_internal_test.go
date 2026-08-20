package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelayHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "regressive first", got: regressiveDelay(1, "3s", "500ms"), want: "3s"},
		{name: "regressive third", got: regressiveDelay(3, "3s", "500ms"), want: "2s"},
		{name: "regressive floor", got: regressiveDelay(42, "3s", "500ms"), want: "0s"},
		{name: "regressive numbers are ms", got: regressiveDelay(2, 3000, 500), want: "2.5s"},
		{name: "regressive broken input", got: regressiveDelay(1, "banana", "500ms"), want: "0s"},
		{name: "backoff first", got: backoffDelay(1, "100ms"), want: "100ms"},
		{name: "backoff fourth", got: backoffDelay(4, "100ms"), want: "800ms"},
		{name: "backoff capped", got: backoffDelay(9, "100ms", "1s"), want: "1s"},
		{name: "jitter degenerate", got: jitterDelay("50ms", "50ms"), want: "50ms"},
		{name: "jitter inverted", got: jitterDelay("250ms", "50ms"), want: "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.got)
		})
	}
}

func TestJitterStaysInRange(t *testing.T) {
	t.Parallel()

	for range 100 {
		got, ok := toDuration(jitterDelay("50ms", "250ms"))
		require.True(t, ok)
		require.GreaterOrEqual(t, got.Milliseconds(), int64(50))
		require.Less(t, got.Milliseconds(), int64(250))
	}
}

func BenchmarkJitterDelay(b *testing.B) {
	for b.Loop() {
		_ = jitterDelay("50ms", "250ms")
	}
}
