package stuber

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestParallelSearchDrainsAllMatchesWithTimes is a regression for the parallel
// search path keeping only ONE best match per 50-stub chunk. Exhausted stubs
// stay in the ranked candidate set (Times is enforced only at reservation), so
// with one-per-chunk collection the search could reserve only the per-chunk
// top stubs and returned "not found" while many matching stubs with spare
// Times budget remained. With full per-chunk collection the parallel path
// drains every matching stub, matching the sequential path.
func TestParallelSearchDrainsAllMatchesWithTimes(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	const n = parallelProcessingThreshold + 20 // force the parallel path (>= 100)

	stubs := make([]*Stub, 0, n)
	for i := range n {
		stubs = append(stubs, &Stub{
			ID:       uuid.New(),
			Service:  "Svc",
			Method:   "M",
			Priority: i, // distinct priorities → deterministic ranking
			Input:    InputData{Equals: map[string]any{"k": "v"}},
			Output:   Output{Data: map[string]any{"i": i}},
			Options:  StubOptions{Times: 1},
		})
	}

	s.upsert(stubs...)

	q := Query{Service: "Svc", Method: "M", Input: []map[string]any{{"k": "v"}}}

	found := 0

	for range n + 5 {
		res, err := s.find(q)
		if err == nil && res.Found() != nil {
			found++
		}
	}

	// All n matching stubs have Times=1, so exactly n reservations must succeed.
	require.Equal(t, n, found, "parallel search must reserve every matching stub, not just per-chunk bests")
}
