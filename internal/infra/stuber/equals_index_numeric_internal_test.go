package stuber

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEqualsIndexFindsMatchesStubForNumericQuery(t *testing.T) {
	t.Parallel()

	for name, queryValue := range map[string]any{
		"json.Number": json.Number("123"),
		"int":         123,
		"int64":       int64(123),
		"float64":     float64(123),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			budgerigar := NewBudgerigar()
			budgerigar.PutMany(&Stub{
				ID:      uuid.New(),
				Service: "svc",
				Method:  "M",
				Input:   InputData{Matches: map[string]any{"id": "^123$"}},
				Output:  Output{Data: map[string]any{"ok": true}},
			})

			result, err := budgerigar.FindByQuery(Query{
				Service: "svc",
				Method:  "M",
				Input:   []map[string]any{{"id": queryValue}},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Found(), "matches stub must be found for a numeric query field")
		})
	}
}

func TestEqualsIndexStillRejectsNonMatchingNumeric(t *testing.T) {
	t.Parallel()

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Input:   InputData{Matches: map[string]any{"id": "^123$"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	})

	result, err := budgerigar.FindByQuery(Query{
		Service: "svc",
		Method:  "M",
		Input:   []map[string]any{{"id": json.Number("456")}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Nil(t, result.Found())
}

func TestEqualsIndexStringStubUnaffected(t *testing.T) {
	t.Parallel()

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Input:   InputData{Equals: map[string]any{"name": "gripmock"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	})

	found, err := budgerigar.FindByQuery(Query{
		Service: "svc",
		Method:  "M",
		Input:   []map[string]any{{"name": "gripmock"}},
	})
	require.NoError(t, err)
	require.NotNil(t, found.Found())

	missing, err := budgerigar.FindByQuery(Query{
		Service: "svc",
		Method:  "M",
		Input:   []map[string]any{{"name": "other"}},
	})
	require.NoError(t, err)
	require.Nil(t, missing.Found())
}
