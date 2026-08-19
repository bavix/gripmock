package stuber

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func bidiMatches(t *testing.T, stubHeaders InputHeader, queryHeaders map[string]any) bool {
	t.Helper()

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Headers: stubHeaders,
		Inputs:  []InputData{{Equals: map[string]any{"deal": "D1"}}},
		Output:  Output{Stream: []any{map[string]any{"ok": true}}},
	})

	result, err := budgerigar.FindByQueryBidi(QueryBidi{Service: "svc", Method: "M", Headers: queryHeaders})
	require.NoError(t, err)

	stub, err := result.Next(map[string]any{"deal": "D1"})

	return err == nil && stub != nil
}

func TestBidiRequiresDeclaredHeaders(t *testing.T) {
	t.Parallel()

	gated := InputHeader{Equals: map[string]any{"x-tier": "gold"}}

	require.True(t, bidiMatches(t, gated, map[string]any{"x-tier": "gold"}))
	require.False(t, bidiMatches(t, gated, map[string]any{"x-tier": "bronze"}))
	require.False(t, bidiMatches(t, gated, nil))
}

func TestBidiWithoutHeaderMatchersIgnoresHeaders(t *testing.T) {
	t.Parallel()

	require.True(t, bidiMatches(t, InputHeader{}, nil))
	require.True(t, bidiMatches(t, InputHeader{}, map[string]any{"x-trace": "abc"}))
}

func TestBidiHeaderMatchersIgnoreUnlistedHeaders(t *testing.T) {
	t.Parallel()

	gated := InputHeader{Equals: map[string]any{"x-tier": "gold"}}

	require.True(t, bidiMatches(t, gated, map[string]any{"x-tier": "gold", "x-trace": "abc"}))
}
