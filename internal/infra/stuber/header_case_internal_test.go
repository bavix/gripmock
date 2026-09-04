package stuber

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func headerStub(t *testing.T, headers InputHeader) (*Budgerigar, uuid.UUID) {
	t.Helper()

	budgerigar := NewBudgerigar()
	id := uuid.New()

	budgerigar.PutMany(&Stub{
		ID:      id,
		Service: "svc",
		Method:  "M",
		Headers: headers,
		Input:   InputData{Equals: map[string]any{"k": "v"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	})

	return budgerigar, id
}

func headerQueryMatches(t *testing.T, budgerigar *Budgerigar, queryHeaders map[string]any) *Stub {
	t.Helper()

	result, err := budgerigar.FindByQuery(Query{
		Service: "svc",
		Method:  "M",
		Headers: queryHeaders,
		Input:   []map[string]any{{"k": "v"}},
	})
	require.NoError(t, err)

	return result.Found()
}

func TestStubHeaderNameIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	budgerigar, id := headerStub(t, InputHeader{Equals: map[string]any{"X-Api-Key": "secret"}})

	found := headerQueryMatches(t, budgerigar, map[string]any{"x-api-key": "secret"})
	require.NotNil(t, found)
	require.Equal(t, id, found.ID)
}

func TestQueryHeaderNameIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	budgerigar, id := headerStub(t, InputHeader{Equals: map[string]any{"x-api-key": "secret"}})

	found := headerQueryMatches(t, budgerigar, map[string]any{"X-Api-Key": "secret"})
	require.NotNil(t, found)
	require.Equal(t, id, found.ID)
}

func TestHeaderValueStaysCaseSensitive(t *testing.T) {
	t.Parallel()

	budgerigar, _ := headerStub(t, InputHeader{Equals: map[string]any{"X-Api-Key": "secret"}})

	require.Nil(t, headerQueryMatches(t, budgerigar, map[string]any{"x-api-key": "SECRET"}))
}

func TestAllHeaderMatcherKindsAreCaseInsensitive(t *testing.T) {
	t.Parallel()

	for name, headers := range map[string]InputHeader{
		"contains": {Contains: map[string]any{"X-Api-Key": "secret"}},
		"matches":  {Matches: map[string]any{"X-Api-Key": "^sec.+$"}},
		"glob":     {Glob: map[string]any{"X-Api-Key": "sec*"}},
		"anyOf":    {AnyOf: []AnyOfHeaderElement{{Equals: map[string]any{"X-Api-Key": "secret"}}}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			budgerigar, id := headerStub(t, headers)

			found := headerQueryMatches(t, budgerigar, map[string]any{"x-api-key": "secret"})
			require.NotNil(t, found)
			require.Equal(t, id, found.ID)
		})
	}
}

func TestBidiHeaderNameIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Headers: InputHeader{Equals: map[string]any{"X-Api-Key": "secret"}},
		Inputs:  []InputData{{Equals: map[string]any{"deal": "D1"}}},
		Output:  Output{Stream: []any{map[string]any{"ok": true}}},
	})

	result, err := budgerigar.FindByQueryBidi(QueryBidi{
		Service: "svc",
		Method:  "M",
		Headers: map[string]any{"x-api-key": "secret"},
	})
	require.NoError(t, err)

	stub, err := result.Next(map[string]any{"deal": "D1"})
	require.NoError(t, err)
	require.NotNil(t, stub)
}

func TestHandlerCandidateHeaderNameIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	stub := &Stub{
		Headers: InputHeader{Equals: map[string]any{"x-api-key": "secret"}},
		Handler: func(_ context.Context, _ any) error { return nil },
	}

	require.True(t, HandlerCandidate(stub, QueryBidi{Headers: map[string]any{"X-Api-Key": "secret"}}))
	require.False(t, HandlerCandidate(stub, QueryBidi{Headers: map[string]any{"X-Api-Key": "other"}}))
}

func TestInspectHeaderNameIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	budgerigar, id := headerStub(t, InputHeader{Equals: map[string]any{"x-api-key": "secret"}})

	report := budgerigar.InspectQuery(Query{
		Service: "svc",
		Method:  "M",
		Headers: map[string]any{"X-Api-Key": "secret"},
		Input:   []map[string]any{{"k": "v"}},
	})

	require.NotNil(t, report.MatchedStubID)
	require.Equal(t, id, *report.MatchedStubID)
}
