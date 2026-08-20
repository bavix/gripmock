package stuber

import (
	"bytes"
	"slices"
)

type rankedMatch struct {
	stub        *Stub
	specificity int
	totalScore  float64
	fieldCount  int
}

type similarCandidate struct {
	stub        *Stub
	score       float64
	specificity int
	fieldCount  int
}

func (s *searcher) buildSimilarCandidateFromRanked(stub *Stub, ranked rankedMatch) similarCandidate {
	return similarCandidate{
		stub:        stub,
		score:       ranked.totalScore,
		specificity: ranked.specificity,
		fieldCount:  ranked.fieldCount,
	}
}

func (s *searcher) scoreWithPriority(query Query, stub *Stub) float64 {
	return s.fastRankV2(query, stub) + float64(stub.Priority)*PriorityMultiplier
}

func betterSimilar(current, candidate similarCandidate) bool {
	if current.stub == nil {
		return true
	}

	if candidate.specificity != current.specificity {
		return candidate.specificity > current.specificity
	}

	if candidate.fieldCount != current.fieldCount {
		return candidate.fieldCount < current.fieldCount
	}

	if candidate.score != current.score {
		return candidate.score > current.score
	}

	// Deterministic tiebreak by stub ID (mirrors compareRankedMatches) so the
	// sequential and parallel paths pick the same Similar stub on a full tie —
	// the parallel path merges chunk candidates in goroutine-completion order.
	return bytes.Compare(candidate.stub.ID[:], current.stub.ID[:]) < 0
}

func (s *searcher) rankedMatchFor(query Query, stub *Stub) rankedMatch {
	return rankedMatch{
		stub:        stub,
		specificity: s.calcSpecificity(stub, query),
		totalScore:  s.scoreWithPriority(query, stub),
		fieldCount:  countStubFields(stub),
	}
}

func compareRankedMatches(a, b rankedMatch) int {
	if a.specificity != b.specificity {
		return b.specificity - a.specificity
	}

	if a.totalScore < b.totalScore {
		return 1
	}

	if a.totalScore > b.totalScore {
		return -1
	}

	if a.fieldCount != b.fieldCount {
		if a.fieldCount > b.fieldCount {
			return -1
		}

		return 1
	}

	// Final deterministic tiebreak by stub ID. Without it fully-tied stubs keep
	// input order, but SortFunc is unstable and the parallel path concatenates
	// chunk matches in goroutine-completion order — so the sequential and
	// parallel searches could reserve DIFFERENT stubs for the same query.
	return bytes.Compare(a.stub.ID[:], b.stub.ID[:])
}

func sortRankedMatches(matches []rankedMatch) {
	slices.SortFunc(matches, compareRankedMatches)
}

func countStubFields(stub *Stub) int {
	count := len(stub.Headers.Equals) + len(stub.Headers.Contains) + len(stub.Headers.Matches) + len(stub.Headers.Glob)

	for _, input := range stub.Matchers() {
		count += len(input.Equals) + len(input.Contains) + len(input.Matches) + len(input.Glob)
		count += countAnyOfFields(input.AnyOf)
	}

	for _, alt := range stub.Headers.AnyOf {
		count += len(alt.Equals) + len(alt.Contains) + len(alt.Matches) + len(alt.Glob)
	}

	return count
}

func countAnyOfFields(anyOf []AnyOfElement) int {
	var n int

	for _, alt := range anyOf {
		n += len(alt.Equals) + len(alt.Contains) + len(alt.Matches) + len(alt.Glob)
	}

	return n
}
