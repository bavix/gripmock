package stuber

import "runtime"

// searchOptimized performs ultra-fast search with minimal allocations.
func (s *searcher) searchOptimized(query Query) (*Result, error) {
	// Internal stubs (the gripmock health status) live in a separate index and
	// are matched ONLY by the gRPC health service, which sets the flag. They are
	// invisible to every user-facing surface (list/get/search/inspect/verify/MCP).
	if query.allowsInternalStubs() {
		if result := s.matchInternalStubs(query); result != nil {
			return result, nil
		}
	}

	// Fast path: ask the equals index for the handful of stubs that could
	// match, instead of walking every stub registered for this service and
	// method. It only ever narrows the set, so a hit here is the same answer
	// the full scan would have produced; anything else falls through below.
	if result := s.searchIndexed(query); result != nil {
		return result, nil
	}

	candidates, err := s.resolveSearchCandidates(query)
	if err != nil {
		return nil, err
	}

	// candidates comes from the pool and is only read during processStubs --
	// the Result keeps *Stub pointers, never the slice itself.
	defer releaseStubs(candidates)

	return s.processStubs(query, candidates)
}

// searchIndexed tries to answer the query from the equals index. It returns a
// result only on a positive match: with no match the "similar" stub reported
// to the user has to be chosen from every registered stub, not just the
// narrowed set, so the caller redoes the search over the full candidate list.
func (s *searcher) searchIndexed(query Query) *Result {
	visible := s.visibleIndexedCandidates(query)
	if len(visible) == 0 {
		return nil
	}

	result, err := s.processStubs(query, visible)
	if err != nil || result == nil || result.Found() == nil {
		return nil
	}

	return result
}

// visibleIndexedCandidates narrows the candidate set through the equals index
// and drops stubs hidden from, or already exhausted for, this session.
func (s *searcher) visibleIndexedCandidates(query Query) []*Stub {
	if len(query.Input) != 1 {
		return nil
	}

	indexes, err := s.storage.posByPN(query.Service, query.Method)
	if err != nil {
		return nil
	}

	candidates, ok := s.storage.indexedCandidates(indexes, query.Input[0])
	if !ok {
		return nil
	}

	visible := candidates[:0]

	for _, stub := range candidates {
		if s.isVisibleAndNotExhausted(stub, query.Session) {
			visible = append(visible, stub)
		}
	}

	return visible
}

// matchInternalStubs searches ONLY the reserved internal storage. It returns a
// result when an internal stub matches, or nil to fall through to user stubs.
func (s *searcher) matchInternalStubs(query Query) *Result {
	if s.internalStorage == nil {
		return nil
	}

	seq, err := s.internalStorage.FindAllAvailable(query.Service, query.Method, query.Session)
	if err != nil {
		return nil
	}

	candidates := collectStubs(seq)
	if len(candidates) == 0 {
		return nil
	}

	result, err := s.processStubs(query, candidates)
	if err != nil || result == nil || result.Found() == nil {
		return nil
	}

	return result
}

func (s *searcher) resolveSearchCandidates(query Query) ([]*Stub, error) {
	lookup := s.lookup(query.Session)

	stubs, err := lookup.LookupServiceAvailable(query.Service, query.Method)
	if err == nil {
		return collectStubsPooled(stubs), nil
	}

	if query.StrictService {
		return nil, ErrStubNotFound
	}

	if !lookup.HasMethodAvailable(query.Method) {
		return nil, ErrStubNotFound
	}

	candidates := collectStubsPooled(lookup.LookupMethodAvailable(query.Method))
	if len(candidates) == 0 {
		releaseStubs(candidates)

		return nil, ErrStubNotFound
	}

	return candidates, nil
}

// processStubs processes the collected stubs with ultra-fast paths.
func (s *searcher) processStubs(query Query, stubs []*Stub) (*Result, error) {
	if len(stubs) == 0 {
		return nil, ErrStubNotFound
	}

	if len(stubs) == 1 {
		return s.processSingleStub(query, stubs[0])
	}

	// Parallel processing for multiple stubs
	if len(stubs) >= parallelProcessingThreshold {
		return s.processStubsParallel(query, stubs)
	}

	// Single-threaded processing for small sets
	return s.processStubsSequential(query, stubs)
}

// processStubsSequential processes stubs sequentially (original logic).
func (s *searcher) processStubsSequential(query Query, stubs []*Stub) (*Result, error) {
	matches := s.collectSequentialCandidates(query, stubs)
	sortRankedMatches(matches)

	// The similar stub is only reported when no match can be reserved, so its
	// (expensive) full-scan search runs lazily.
	return s.resultFromRankedMatches(query, matches, func() *Stub {
		return s.bestSimilarCandidate(query, stubs).stub
	})
}

func (s *searcher) processSingleStub(query Query, stub *Stub) (*Result, error) {
	if s.fastMatchV2(query, stub) && s.tryReserve(query, stub) {
		return &Result{found: stub}, nil
	}

	return &Result{similar: stub}, nil
}

// collectSequentialCandidates ranks the stubs matching the query.
//
// Ranking a stub (specificity + score + field count) is far more expensive than
// testing whether it matches at all, and the ranking of a non-matching stub is
// only ever read to choose the `similar` stub reported when nothing matched.
// So the cheap boolean test runs over every candidate here, and ranking runs
// only over the (normally tiny) matched subset; the similar-stub pass is
// deferred to bestSimilarCandidate and skipped entirely on the happy path.
// With a large stub set under one service/method this avoids scoring hundreds
// of thousands of stubs on every successful request.
func (s *searcher) collectSequentialCandidates(query Query, stubs []*Stub) []rankedMatch {
	var matches []rankedMatch

	for _, stub := range stubs {
		if s.fastMatchV2(query, stub) {
			matches = append(matches, s.rankedMatchFor(query, stub))
		}
	}

	return matches
}

// bestSimilarCandidate scores every stub to find the closest non-matching one.
// Only reached when no stub matched the query (the error path).
func (s *searcher) bestSimilarCandidate(query Query, stubs []*Stub) similarCandidate {
	var best similarCandidate

	for _, stub := range stubs {
		candidate := s.buildSimilarCandidateFromRanked(stub, s.rankedMatchFor(query, stub))

		if betterSimilar(best, candidate) {
			best = candidate
		}
	}

	return best
}

func (s *searcher) reserveFirstRankedMatch(query Query, matches []rankedMatch) *Stub {
	for _, match := range matches {
		if s.tryReserve(query, match.stub) {
			return match.stub
		}
	}

	return nil
}

// resultFromRankedMatches reserves the best ranked match, falling back to the
// closest similar stub. similarFn is only invoked on that fallback path, so
// callers can skip computing it while a match looks reservable.
func (s *searcher) resultFromRankedMatches(query Query, matches []rankedMatch, similarFn func() *Stub) (*Result, error) {
	if found := s.reserveFirstRankedMatch(query, matches); found != nil {
		return &Result{found: found}, nil
	}

	if similar := similarFn(); similar != nil {
		return &Result{similar: similar}, nil
	}

	return nil, ErrStubNotFound
}

// processStubsParallel processes stubs in parallel using goroutines. The
// worker count is capped at GOMAXPROCS: a fixed small chunk size (e.g. 50)
// spawns one goroutine per chunk regardless of total candidate count, which
// at large stub counts (hundreds of thousands) fans out into thousands of
// goroutines per request -- goroutine/channel overhead then dominates over
// the actual matching work, especially under concurrent requests competing
// for the same limited CPUs.
func (s *searcher) processStubsParallel(query Query, stubs []*Stub) (*Result, error) {
	chunkSize := parallelChunkSize(len(stubs))

	numChunks := (len(stubs) + chunkSize - 1) / chunkSize
	results := make(chan []rankedMatch, numChunks)

	for i := range numChunks {
		start := i * chunkSize

		end := min(start+chunkSize, len(stubs))
		go func(chunkStubs []*Stub) {
			results <- s.collectSequentialCandidates(query, chunkStubs)
		}(stubs[start:end])
	}

	bestMatches := collectChunkResults(results, numChunks)
	sortRankedMatches(bestMatches)

	// Same lazy similar-stub search as the sequential path.
	return s.resultFromRankedMatches(query, bestMatches, func() *Stub {
		return s.bestSimilarCandidate(query, stubs).stub
	})
}

// parallelChunkSize returns a chunk size that caps the number of goroutines
// spawned by processStubsParallel at GOMAXPROCS, so goroutine count scales
// with available CPUs instead of candidate count.
func parallelChunkSize(numStubs int) int {
	workers := max(runtime.GOMAXPROCS(0), 1)
	chunkSize := (numStubs + workers - 1) / workers

	return max(chunkSize, 1)
}

func collectChunkResults(results chan []rankedMatch, numChunks int) []rankedMatch {
	// Matches are normally a tiny fraction of the candidates, so start empty
	// and let append size it; pre-allocating per chunk would over-allocate.
	bestMatches := make([]rankedMatch, 0, numChunks)

	for range numChunks {
		bestMatches = append(bestMatches, <-results...)
	}

	return bestMatches
}
