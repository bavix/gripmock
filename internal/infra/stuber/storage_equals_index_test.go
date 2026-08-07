package stuber_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// The equals index narrows candidates before the matcher runs, so every case
// below asserts that the narrowed search still returns exactly what a full
// scan would: the index must never hide a stub that would have matched.

// message pulls the "message" field out of a stub's output payload.
func message(t *testing.T, stub *stuber.Stub) string {
	t.Helper()

	data, ok := stub.Output.Data.(map[string]any)
	require.True(t, ok, "output data is not a map")

	value, ok := data["message"].(string)
	require.True(t, ok, "output has no string message")

	return value
}

func indexQuery(input map[string]any) stuber.Query {
	return stuber.Query{Service: "Greeter", Method: "SayHello", Input: []map[string]any{input}}
}

func equalsStub(value string, message string) *stuber.Stub {
	return &stuber.Stub{
		ID:      uuid.New(),
		Service: "Greeter",
		Method:  "SayHello",
		Input:   stuber.InputData{Equals: map[string]any{"name": value}},
		Output:  stuber.Output{Data: map[string]any{"message": message}},
	}
}

// fill adds `count` indexable stubs so searches go through the index rather
// than any small-set shortcut.
func fill(s *stuber.Budgerigar, count int) {
	stubs := make([]*stuber.Stub, count)
	for i := range stubs {
		stubs[i] = equalsStub(fmt.Sprintf("filler-%d", i), "filler")
	}

	s.PutMany(stubs...)
}

func TestEqualsIndexFindsStubAmongMany(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 500)
	s.PutMany(equalsStub("needle", "found"))

	result, err := s.FindByQuery(indexQuery(map[string]any{"name": "needle"}))
	require.NoError(t, err)
	require.NotNil(t, result.Found())
	require.Equal(t, "found", message(t, result.Found()))
}

func TestEqualsIndexMissFallsBackToSimilar(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 500)

	// No stub matches, so the search must fall through to the full scan and
	// still report a "similar" stub rather than an empty result.
	result, err := s.FindByQuery(indexQuery(map[string]any{"name": "absent"}))
	require.NoError(t, err)
	require.Nil(t, result.Found())
	require.NotNil(t, result.Similar())
}

func TestEqualsIndexMultiFieldStub(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 500)
	s.PutMany(&stuber.Stub{
		ID:      uuid.New(),
		Service: "Greeter",
		Method:  "SayHello",
		Input:   stuber.InputData{Equals: map[string]any{"name": "ann", "city": "berlin"}},
		Output:  stuber.Output{Data: map[string]any{"message": "multi"}},
	})

	// Both fields present -> match.
	result, err := s.FindByQuery(indexQuery(map[string]any{"name": "ann", "city": "berlin"}))
	require.NoError(t, err)
	require.NotNil(t, result.Found())
	require.Equal(t, "multi", message(t, result.Found()))

	// Only one field present -> the stub requires both, so no match.
	result, err = s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.NoError(t, err)
	require.Nil(t, result.Found())
}

func TestEqualsIndexKeyCaseVariations(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ stubKey, queryKey string }{
		{"user_name", "userName"},
		{"userName", "user_name"},
		{"user_name", "user_name"},
	} {
		t.Run(tc.stubKey+"/"+tc.queryKey, func(t *testing.T) {
			t.Parallel()

			s := newBudgerigar()
			fill(s, 200)
			s.PutMany(&stuber.Stub{
				ID:      uuid.New(),
				Service: "Greeter",
				Method:  "SayHello",
				Input:   stuber.InputData{Equals: map[string]any{tc.stubKey: "ann"}},
				Output:  stuber.Output{Data: map[string]any{"message": "cased"}},
			})

			result, err := s.FindByQuery(indexQuery(map[string]any{tc.queryKey: "ann"}))
			require.NoError(t, err)
			require.NotNil(t, result.Found())
			require.Equal(t, "cased", message(t, result.Found()))
		})
	}
}

func TestEqualsIndexNonIndexableStubsStillMatch(t *testing.T) {
	t.Parallel()

	// None of these can be indexed (non-string value, or a matcher kind the
	// index cannot reason about), so each must still be found via the scan
	// even while indexable stubs dominate the same service/method.
	for name, stub := range map[string]*stuber.Stub{
		"numeric equals": {
			Input:  stuber.InputData{Equals: map[string]any{"age": 42}},
			Output: stuber.Output{Data: map[string]any{"message": "hit"}},
		},
		"contains": {
			Input:  stuber.InputData{Contains: map[string]any{"name": "ann"}},
			Output: stuber.Output{Data: map[string]any{"message": "hit"}},
		},
		"matches": {
			Input:  stuber.InputData{Matches: map[string]any{"name": "^an+$"}},
			Output: stuber.Output{Data: map[string]any{"message": "hit"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := newBudgerigar()
			fill(s, 300)

			stub.ID = uuid.New()
			stub.Service = "Greeter"
			stub.Method = "SayHello"
			s.PutMany(stub)

			input := map[string]any{"name": "ann"}
			if name == "numeric equals" {
				input = map[string]any{"age": 42}
			}

			result, err := s.FindByQuery(indexQuery(input))
			require.NoError(t, err)
			require.NotNil(t, result.Found())
			require.Equal(t, "hit", message(t, result.Found()))
		})
	}
}

func TestEqualsIndexRespectsPriority(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 300)

	low := equalsStub("ann", "low")
	high := equalsStub("ann", "high")
	high.Priority = 10
	s.PutMany(low, high)

	result, err := s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.NoError(t, err)
	require.NotNil(t, result.Found())
	require.Equal(t, "high", message(t, result.Found()))
}

func TestEqualsIndexRespectsTimesLimit(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 300)

	stub := equalsStub("ann", "once")
	stub.Options = stuber.StubOptions{Times: 1}
	s.PutMany(stub)

	result, err := s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.NoError(t, err)
	require.NotNil(t, result.Found())

	// Exhausted: the index must not keep serving it.
	result, err = s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.NoError(t, err)
	require.Nil(t, result.Found())
}

func TestEqualsIndexUpdatedOnDelete(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 300)

	stub := equalsStub("ann", "present")
	ids := s.PutMany(stub)

	result, err := s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.NoError(t, err)
	require.NotNil(t, result.Found())

	require.Equal(t, 1, s.DeleteByID(ids...))

	result, err = s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.NoError(t, err)
	require.Nil(t, result.Found())
}

func TestEqualsIndexUpdatedOnValueChange(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 300)

	stub := equalsStub("before", "v1")
	s.PutMany(stub)

	updated := &stuber.Stub{
		ID:      stub.ID,
		Service: "Greeter",
		Method:  "SayHello",
		Input:   stuber.InputData{Equals: map[string]any{"name": "after"}},
		Output:  stuber.Output{Data: map[string]any{"message": "v2"}},
	}
	s.UpdateMany(updated)

	// The old value must no longer resolve...
	result, err := s.FindByQuery(indexQuery(map[string]any{"name": "before"}))
	require.NoError(t, err)
	require.Nil(t, result.Found())

	// ...and the new one must.
	result, err = s.FindByQuery(indexQuery(map[string]any{"name": "after"}))
	require.NoError(t, err)
	require.NotNil(t, result.Found())
	require.Equal(t, "v2", message(t, result.Found()))
}

func TestEqualsIndexSessionIsolation(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 300)

	scoped := equalsStub("ann", "scoped")
	scoped.Session = "s1"
	s.PutMany(scoped)

	// Visible inside its own session.
	query := indexQuery(map[string]any{"name": "ann"})
	query.Session = "s1"
	result, err := s.FindByQuery(query)
	require.NoError(t, err)
	require.NotNil(t, result.Found())
	require.Equal(t, "scoped", message(t, result.Found()))

	// Hidden from the global scope.
	result, err = s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.NoError(t, err)
	require.Nil(t, result.Found())
}

func TestEqualsIndexClearEmptiesIt(t *testing.T) {
	t.Parallel()

	s := newBudgerigar()
	fill(s, 300)
	s.PutMany(equalsStub("ann", "present"))
	s.Clear()

	_, err := s.FindByQuery(indexQuery(map[string]any{"name": "ann"}))
	require.Error(t, err)
}
