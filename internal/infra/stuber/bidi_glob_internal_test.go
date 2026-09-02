package stuber

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func bidiGlobMatches(t *testing.T, input InputData, messageData map[string]any) bool {
	t.Helper()

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Inputs:  []InputData{input},
		Output:  Output{Stream: []any{map[string]any{"ok": true}}},
	})

	result, err := budgerigar.FindByQueryBidi(QueryBidi{Service: "svc", Method: "M"})
	require.NoError(t, err)

	stub, err := result.Next(messageData)

	return err == nil && stub != nil
}

func TestBidiInputGlobIsEnforced(t *testing.T) {
	t.Parallel()

	globOnly := InputData{Glob: map[string]any{"name": "abc*"}}

	require.True(t, bidiGlobMatches(t, globOnly, map[string]any{"name": "abcdef"}))
	require.False(t, bidiGlobMatches(t, globOnly, map[string]any{"name": "zzz"}))
	require.False(t, bidiGlobMatches(t, globOnly, map[string]any{"other": "abcdef"}))
}

func TestBidiInputGlobCombinesWithEquals(t *testing.T) {
	t.Parallel()

	combined := InputData{
		Equals: map[string]any{"deal": "D1"},
		Glob:   map[string]any{"name": "abc*"},
	}

	require.True(t, bidiGlobMatches(t, combined, map[string]any{"deal": "D1", "name": "abcdef"}))
	require.False(t, bidiGlobMatches(t, combined, map[string]any{"deal": "D1", "name": "zzz"}))
}

func TestBidiInputGlobInsideAnyOf(t *testing.T) {
	t.Parallel()

	anyOf := InputData{
		AnyOf: []AnyOfElement{
			{Glob: map[string]any{"name": "abc*"}},
			{Equals: map[string]any{"deal": "D9"}},
		},
	}

	require.True(t, bidiGlobMatches(t, anyOf, map[string]any{"name": "abcdef"}))
	require.True(t, bidiGlobMatches(t, anyOf, map[string]any{"deal": "D9"}))
	require.False(t, bidiGlobMatches(t, anyOf, map[string]any{"name": "zzz"}))
}

func TestBidiGlobParticipatesInRanking(t *testing.T) {
	t.Parallel()

	budgerigar := NewBudgerigar()
	globID := uuid.New()
	exactID := uuid.New()

	budgerigar.PutMany(
		&Stub{
			ID:      globID,
			Service: "svc",
			Method:  "M",
			Inputs:  []InputData{{Glob: map[string]any{"name": "abc*"}}},
			Output:  Output{Stream: []any{map[string]any{"who": "glob"}}},
		},
		&Stub{
			ID:      exactID,
			Service: "svc",
			Method:  "M",
			Inputs:  []InputData{{Glob: map[string]any{"name": "abc*"}, Equals: map[string]any{"name": "abcdef"}}},
			Output:  Output{Stream: []any{map[string]any{"who": "exact"}}},
		},
	)

	result, err := budgerigar.FindByQueryBidi(QueryBidi{Service: "svc", Method: "M"})
	require.NoError(t, err)

	stub, err := result.Next(map[string]any{"name": "abcdef"})
	require.NoError(t, err)
	require.Equal(t, exactID, stub.ID)
}
