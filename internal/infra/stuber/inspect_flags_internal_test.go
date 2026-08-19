package stuber

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func inspectGatedStub(t *testing.T, query Query) InspectCandidate {
	t.Helper()

	id := uuid.New()
	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      id,
		Service: "svc",
		Method:  "M",
		Headers: InputHeader{Equals: map[string]any{"x-tenant": "acme"}},
		Input:   InputData{Equals: map[string]any{"id": "1"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	})

	query.Service = "svc"
	query.Method = "M"

	report := budgerigar.InspectQuery(query)

	for _, candidate := range report.Candidates {
		if candidate.ID == id {
			return candidate
		}
	}

	t.Fatal("the stub was not reported as a candidate")

	return InspectCandidate{}
}

func TestInspectBlamesOnlyTheHeaders(t *testing.T) {
	t.Parallel()

	candidate := inspectGatedStub(t, Query{Input: []map[string]any{{"id": "1"}}})

	require.False(t, candidate.Matched)
	require.False(t, candidate.HeadersMatched)
	require.True(t, candidate.InputMatched, "the body did match; only the header was missing")
	require.Contains(t, candidate.ExcludedBy, "headers")
	require.NotContains(t, candidate.ExcludedBy, "input")
}

func TestInspectBlamesOnlyTheInput(t *testing.T) {
	t.Parallel()

	candidate := inspectGatedStub(t, Query{
		Headers: map[string]any{"x-tenant": "acme"},
		Input:   []map[string]any{{"id": "999"}},
	})

	require.False(t, candidate.Matched)
	require.True(t, candidate.HeadersMatched)
	require.False(t, candidate.InputMatched)
	require.Contains(t, candidate.ExcludedBy, "input")
	require.NotContains(t, candidate.ExcludedBy, "headers")
}

func TestInspectBlamesBothWhenBothDiffer(t *testing.T) {
	t.Parallel()

	candidate := inspectGatedStub(t, Query{
		Headers: map[string]any{"x-tenant": "other"},
		Input:   []map[string]any{{"id": "999"}},
	})

	require.False(t, candidate.Matched)
	require.False(t, candidate.HeadersMatched)
	require.False(t, candidate.InputMatched)
	require.Contains(t, candidate.ExcludedBy, "headers")
	require.Contains(t, candidate.ExcludedBy, "input")
}

func TestInspectReportsAFullMatch(t *testing.T) {
	t.Parallel()

	candidate := inspectGatedStub(t, Query{
		Headers: map[string]any{"x-tenant": "acme"},
		Input:   []map[string]any{{"id": "1"}},
	})

	require.True(t, candidate.Matched)
	require.True(t, candidate.HeadersMatched)
	require.True(t, candidate.InputMatched)
	require.Empty(t, candidate.ExcludedBy)
}
