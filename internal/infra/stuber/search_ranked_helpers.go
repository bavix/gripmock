package stuber

import "slices"

type rankedMatch struct {
	stub        *Stub
	specificity int
	totalScore  float64
	fieldCount  int
}

type chunkOutcome struct {
	bestMatch   rankedMatch
	mostSimilar scoredStub
}

type scoredStub struct {
	stub  *Stub
	score float64
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
		fieldCount:  s.ranker.FieldCount(stub),
	}
}

func (s *searcher) scoreWithPriority(query Query, stub *Stub) float64 {
	return s.ranker.Score(query, stub) + float64(stub.Priority)*PriorityMultiplier
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

	return candidate.score > current.score
}

func (s *searcher) rankedMatchFor(query Query, stub *Stub) rankedMatch {
	return rankedMatch{
		stub:        stub,
		specificity: s.ranker.Specificity(query, stub),
		totalScore:  s.scoreWithPriority(query, stub),
		fieldCount:  s.ranker.FieldCount(stub),
	}
}

func (s *searcher) evaluateRankedMatch(query Query, stub *Stub) (rankedMatch, bool) {
	ranked := s.rankedMatchFor(query, stub)

	return ranked, s.matcher.Match(query, stub)
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

	return 0
}

func betterRanked(current, candidate rankedMatch) bool {
	if current.stub == nil {
		return true
	}

	return compareRankedMatches(candidate, current) < 0
}

func sortRankedMatches(matches []rankedMatch) {
	slices.SortFunc(matches, compareRankedMatches)
}

func pickHigherScore(current, candidate scoredStub) scoredStub {
	if current.stub == nil || candidate.score > current.score {
		return candidate
	}

	return current
}
