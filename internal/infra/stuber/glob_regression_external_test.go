package stuber_test

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Regression: globMatch looked up actualMap[key] directly, so a glob keyed
// user_id would not match a query field serialized as userId — unlike equals.
func TestGlobMatcherKeyVariations(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	id := uuid.New()
	s.PutMany(&stuber.Stub{
		ID: id, Service: "s", Method: "m",
		Input:  stuber.InputData{Glob: map[string]any{"user_id": "a*"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	})

	// camelCase query field must resolve to the snake_case glob key.
	res, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"userId": "alex"}}})
	require.NoError(t, err)
	require.NotNil(t, res.Found())
	require.Equal(t, id, res.Found().ID)

	// Non-matching glob pattern.
	res2, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"userId": "bob"}}})
	require.NoError(t, err)
	require.Nil(t, res2.Found())
}

// Regression: fastMatchStream's condition check ignored Glob, so a stream stub
// whose element had only a glob matcher was treated as condition-less and never
// matched.
func TestGlobOnlyStreamStubMatches(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	id := uuid.New()
	s.PutMany(&stuber.Stub{
		ID: id, Service: "s", Method: "m",
		Inputs: []stuber.InputData{{Glob: map[string]any{"name": "h*"}}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	})

	res, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"name": "hello"}}})
	require.NoError(t, err)
	require.NotNil(t, res.Found(), "glob-only stream stub must match")
	require.Equal(t, id, res.Found().ID)
}

// Regression: fastMatchInput's single-matcher fast paths (equals/contains/matches
// only) skipped the Glob constraint, so a unary stub combining Equals with Glob
// matched requests that violated the glob.
func TestCombinedEqualsGlobRejectsNonMatchingGlob(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	id := uuid.New()
	s.PutMany(&stuber.Stub{
		ID: id, Service: "s", Method: "m",
		Input:  stuber.InputData{Equals: map[string]any{"a": "1"}, Glob: map[string]any{"b": "x*"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	})

	hit, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"a": "1", "b": "xyz"}}})
	require.NoError(t, err)
	require.NotNil(t, hit.Found())
	require.Equal(t, id, hit.Found().ID)

	miss, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"a": "1", "b": "zzz"}}})
	require.NoError(t, err)
	require.Nil(t, miss.Found(), "equals+glob stub must reject a value failing the glob")
}

// Regression: betterSimilar had no final tiebreak, so on a full tie the Similar
// (closest-non-match) diagnostic diverged between the sequential and parallel
// paths. The ID tiebreak makes it deterministic — the smallest stub ID wins.
func TestTiedSimilarResolvesDeterministicallyParallel(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()

	const n = 150

	minID := uuid.UUID{}

	stubs := make([]*stuber.Stub, 0, n)
	for i := range n {
		id := uuid.New()
		if i == 0 || bytes.Compare(id[:], minID[:]) < 0 {
			minID = id
		}

		stubs = append(stubs, &stuber.Stub{
			ID: id, Service: "s", Method: "m",
			Input:  stuber.InputData{Equals: map[string]any{"k": "v"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		})
	}

	s.PutMany(stubs...)

	for range 5 {
		res, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"k": "nomatch"}}})
		require.NoError(t, err)
		require.Nil(t, res.Found())
		require.NotNil(t, res.Similar())
		require.Equal(t, minID, res.Similar().ID, "tied Similar must resolve to the smallest stub ID")
	}
}

// Boundary cases for List pagination: offset past the end, zero limit, and the
// exact-boundary offset must all behave and report the correct pre-page total.
func TestListPaginationBoundaries(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	for range 5 {
		s.PutMany(&stuber.Stub{ID: uuid.New(), Service: "s", Method: "m", Input: stuber.InputData{Equals: map[string]any{"k": "v"}}})
	}

	// limit 0 → all rows, total 5.
	all, total := s.List(stuber.ListOptions{})
	require.Equal(t, 5, total)
	require.Len(t, all, 5)

	// offset == total → empty page, total still 5.
	page, total := s.List(stuber.ListOptions{Offset: 5, Limit: 2})
	require.Equal(t, 5, total)
	require.Empty(t, page)

	// offset past the end → empty page, correct total.
	page, total = s.List(stuber.ListOptions{Offset: 999, Limit: 2})
	require.Equal(t, 5, total)
	require.Empty(t, page)

	// limit larger than remaining → clamped.
	page, total = s.List(stuber.ListOptions{Offset: 4, Limit: 10})
	require.Equal(t, 5, total)
	require.Len(t, page, 1)
}

// Regression: sortStubs had no total-order tiebreak, so stubs tying on the sort
// key kept values()'s random map iteration order — paging duplicated/dropped
// rows across independently-sorted page requests.
func TestListPaginationDeterministicAcrossPages(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()

	const total = 40
	for range total {
		s.PutMany(&stuber.Stub{ID: uuid.New(), Service: "s", Method: "m", Input: stuber.InputData{Equals: map[string]any{"k": "v"}}})
	}

	const pageSize = 10

	seen := map[uuid.UUID]int{}

	for offset := 0; offset < total; offset += pageSize {
		page, gotTotal := s.List(stuber.ListOptions{Limit: pageSize, Offset: offset})
		require.Equal(t, total, gotTotal)

		for _, st := range page {
			seen[st.ID]++
		}
	}

	require.Len(t, seen, total, "every stub must appear exactly once across all pages")

	for id, n := range seen {
		require.Equalf(t, 1, n, "stub %s appeared %d times across pages", id, n)
	}
}

// A matcher-kind filter that matches nothing yields an empty result, and a
// headers-only stub (no input matcher) is excluded from input-matcher filters.
func TestListMatcherFilterEdgeCases(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	s.PutMany(
		&stuber.Stub{ID: uuid.New(), Service: "s", Method: "m", Input: stuber.InputData{Equals: map[string]any{"a": 1}}},
		&stuber.Stub{ID: uuid.New(), Service: "s", Method: "m", Headers: stuber.InputHeader{Equals: map[string]any{"h": "1"}}},
	)

	// No input stub declares glob.
	_, globTotal := s.List(stuber.ListOptions{Matchers: []string{"glob"}})
	require.Equal(t, 0, globTotal)

	// equals matches only the input-equals stub, not the headers-only one.
	_, eqTotal := s.List(stuber.ListOptions{Matchers: []string{"equals"}})
	require.Equal(t, 1, eqTotal)
}

// List's Matchers filter keeps only stubs declaring the requested matcher
// kind(s), with OR semantics across kinds.
func TestListMatcherKindFilter(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	eqStub := &stuber.Stub{ID: uuid.New(), Service: "s", Method: "m", Input: stuber.InputData{Equals: map[string]any{"a": 1}}}
	globStub := &stuber.Stub{ID: uuid.New(), Service: "s", Method: "m", Input: stuber.InputData{Glob: map[string]any{"g": "x*"}}}
	anyStub := &stuber.Stub{
		ID: uuid.New(), Service: "s", Method: "m",
		Input: stuber.InputData{AnyOf: []stuber.AnyOfElement{{Equals: map[string]any{"a": 1}}}},
	}
	s.PutMany(eqStub, globStub, anyStub)

	glob, globTotal := s.List(stuber.ListOptions{Matchers: []string{"glob"}})
	require.Equal(t, 1, globTotal)
	require.Len(t, glob, 1)
	require.Equal(t, globStub.ID, glob[0].ID)

	// OR across kinds: glob OR anyOf → the glob and anyOf stubs.
	_, orTotal := s.List(stuber.ListOptions{Matchers: []string{"glob", "anyOf"}})
	require.Equal(t, 2, orTotal)

	// No filter → all stubs.
	_, allTotal := s.List(stuber.ListOptions{})
	require.Equal(t, 3, allTotal)
}

// Regression: fastMatchStream's condition check omitted AnyOf (same class as the
// glob gap), so a stream stub element carrying only an anyOf matcher was treated
// as condition-less and never matched.
func TestAnyOfOnlyStreamStubMatches(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	id := uuid.New()
	s.PutMany(&stuber.Stub{
		ID: id, Service: "s", Method: "m",
		Inputs: []stuber.InputData{{AnyOf: []stuber.AnyOfElement{
			{Equals: map[string]any{"name": "alice"}},
			{Equals: map[string]any{"name": "bob"}},
		}}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	})

	res, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"name": "bob"}}})
	require.NoError(t, err)
	require.NotNil(t, res.Found(), "anyOf-only stream stub must match")
	require.Equal(t, id, res.Found().ID)

	// A value in neither alternative must not match.
	miss, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"name": "zoe"}}})
	require.NoError(t, err)
	require.Nil(t, miss.Found())
}

// Regression: compareRankedMatches returned 0 for fully-tied stubs with no final
// tiebreak. With an unstable sort and the parallel path assembling matches in
// goroutine-completion order, the winner was nondeterministic and could differ
// between the sequential and parallel searches. The ID tiebreak makes the choice
// deterministic — the smallest stub ID always wins a tie.
func TestTiedStubsResolveDeterministicallyParallel(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()

	// 150 fully-identical stubs (> parallelProcessingThreshold of 100) so the
	// parallel path runs. Unlimited Times means reservation never consumes them.
	const n = 150

	minID := uuid.UUID{}

	stubs := make([]*stuber.Stub, 0, n)
	for i := range n {
		id := uuid.New()
		if i == 0 || bytes.Compare(id[:], minID[:]) < 0 {
			minID = id
		}

		stubs = append(stubs, &stuber.Stub{
			ID: id, Service: "s", Method: "m",
			Input:  stuber.InputData{Equals: map[string]any{"k": "v"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		})
	}

	s.PutMany(stubs...)

	// Every query must reserve the same (smallest-ID) stub.
	for range 5 {
		res, err := s.FindByQuery(stuber.Query{Service: "s", Method: "m", Input: []map[string]any{{"k": "v"}}})
		require.NoError(t, err)
		require.NotNil(t, res.Found())
		require.Equal(t, minID, res.Found().ID, "tie must resolve to the smallest stub ID deterministically")
	}
}
