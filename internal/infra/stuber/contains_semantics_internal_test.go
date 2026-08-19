package stuber

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func matchesQuery(t *testing.T, stub *Stub, query Query) bool {
	t.Helper()

	budgerigar := NewBudgerigar()
	stub.ID = uuid.New()
	stub.Service = "svc"
	stub.Method = "M"
	stub.Output = Output{Data: map[string]any{"ok": true}}
	budgerigar.PutMany(stub)

	query.Service = "svc"
	query.Method = "M"

	result, err := budgerigar.FindByQuery(query)

	return err == nil && result != nil && result.Found() != nil
}

func TestContainsOnStringsRequiresEquality(t *testing.T) {
	t.Parallel()

	stub := func() *Stub {
		return &Stub{Input: InputData{Contains: map[string]any{"name": "grip"}}}
	}

	require.True(t, matchesQuery(t, stub(), Query{Input: []map[string]any{{"name": "grip"}}}))
	require.False(t, matchesQuery(t, stub(), Query{Input: []map[string]any{{"name": "gripmock"}}}))
}

func TestContainsIgnoresUnlistedFields(t *testing.T) {
	t.Parallel()

	stub := &Stub{Input: InputData{Contains: map[string]any{"name": "grip"}}}

	require.True(t, matchesQuery(t, stub, Query{
		Input: []map[string]any{{"name": "grip", "extra": "ignored"}},
	}))
}

func TestContainsOnArraysIgnoresOrder(t *testing.T) {
	t.Parallel()

	stub := func() *Stub {
		return &Stub{Input: InputData{Contains: map[string]any{"tags": []any{"a"}}}}
	}

	require.True(t, matchesQuery(t, stub(), Query{Input: []map[string]any{{"tags": []any{"b", "a"}}}}))
	require.False(t, matchesQuery(t, stub(), Query{Input: []map[string]any{{"tags": []any{"b"}}}}))
}

func TestContainsOnNestedObjectsIsRecursive(t *testing.T) {
	t.Parallel()

	stub := &Stub{Input: InputData{Contains: map[string]any{"user": map[string]any{"id": "7"}}}}

	require.True(t, matchesQuery(t, stub, Query{
		Input: []map[string]any{{"user": map[string]any{"id": "7", "name": "ann"}}},
	}))
}

func TestHeaderContainsRequiresEqualValues(t *testing.T) {
	t.Parallel()

	stub := func() *Stub {
		return &Stub{
			Headers: InputHeader{Contains: map[string]any{"authorization": "Bearer"}},
			Input:   InputData{Equals: map[string]any{"a": "1"}},
		}
	}

	body := []map[string]any{{"a": "1"}}

	require.True(t, matchesQuery(t, stub(), Query{
		Headers: map[string]any{"authorization": "Bearer"},
		Input:   body,
	}))
	require.False(t, matchesQuery(t, stub(), Query{
		Headers: map[string]any{"authorization": "Bearer token-123"},
		Input:   body,
	}))
}

func TestHeaderContainsIgnoresUnlistedHeaders(t *testing.T) {
	t.Parallel()

	stub := &Stub{
		Headers: InputHeader{Contains: map[string]any{"x-tenant": "acme"}},
		Input:   InputData{Equals: map[string]any{"a": "1"}},
	}

	require.True(t, matchesQuery(t, stub, Query{
		Headers: map[string]any{"x-tenant": "acme", "x-trace": "abc"},
		Input:   []map[string]any{{"a": "1"}},
	}))
}
