package stuber

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// The stub loader decodes numbers as json.Number, while encoding/json yields
// float64. Reading only one of them silently dropped output.stream error codes
// and every marked element failed with the default Aborted.
func TestExtractGripMockErrorAcceptsEveryNumberShape(t *testing.T) {
	t.Parallel()

	for name, code := range map[string]any{
		"json.Number": json.Number("8"),
		"float64":     float64(8),
		"int":         8,
		"int64":       int64(8),
		"uint32":      uint32(8),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			el := ExtractGripMock(map[string]any{
				GripMockKey: map[string]any{"error": "boom", "code": code},
			})

			require.True(t, el.HasError)
			require.NotNil(t, el.Code, "code must survive parsing")
			require.Equal(t, codes.ResourceExhausted, *el.Code)
		})
	}
}

func TestExtractGripMockStripsKeyEvenWhenMalformed(t *testing.T) {
	t.Parallel()

	m := map[string]any{GripMockKey: "not-a-map", "field": 1}
	el := ExtractGripMock(m)

	require.False(t, el.HasError)
	require.False(t, el.HasDelay)
	require.NotContains(t, m, GripMockKey, "the marker must never reach protobuf unmarshalling")
}

func TestExtractGripMockReadsDelayAndErrorTogether(t *testing.T) {
	t.Parallel()

	el := ExtractGripMock(map[string]any{
		GripMockKey: map[string]any{"delay": "10ms", "error": "boom"},
	})

	require.True(t, el.HasDelay)
	require.True(t, el.HasError)
	require.Nil(t, el.Code, "no code means the caller falls back to the default status")
}
