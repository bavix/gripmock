package stuber

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

	candidates, err := s.resolveSearchCandidates(query)
	if err != nil {
		return nil, err
	}

	return s.processStubs(query, candidates)
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
		return collectStubs(stubs), nil
	}

	if query.StrictService {
		return nil, ErrStubNotFound
	}

	if !lookup.HasMethodAvailable(query.Method) {
		return nil, ErrStubNotFound
	}

	candidates := collectStubs(lookup.LookupMethodAvailable(query.Method))
	if len(candidates) == 0 {
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
	matches, best := s.collectSequentialCandidates(query, stubs)
	sortRankedMatches(matches)

	return s.resultFromRankedAndSimilar(query, matches, best.stub)
}

func (s *searcher) processSingleStub(query Query, stub *Stub) (*Result, error) {
	if s.fastMatchV2(query, stub) && s.tryReserve(query, stub) {
		return &Result{found: stub}, nil
	}

	return &Result{similar: stub}, nil
}

func (s *searcher) collectSequentialCandidates(query Query, stubs []*Stub) ([]rankedMatch, similarCandidate) {
	var (
		matches []rankedMatch
		best    similarCandidate
	)

	for _, stub := range stubs {
		ranked, matched := s.evaluateRankedMatch(query, stub)

		candidate := s.buildSimilarCandidateFromRanked(stub, ranked)
		if matched {
			matches = append(matches, ranked)
		}

		if betterSimilar(best, candidate) {
			best = candidate
		}
	}

	return matches, best
}

func (s *searcher) reserveFirstRankedMatch(query Query, matches []rankedMatch) *Stub {
	for _, match := range matches {
		if s.tryReserve(query, match.stub) {
			return match.stub
		}
	}

	return nil
}

func (s *searcher) resultFromRankedAndSimilar(query Query, matches []rankedMatch, similar *Stub) (*Result, error) {
	if found := s.reserveFirstRankedMatch(query, matches); found != nil {
		return &Result{found: found}, nil
	}

	if similar != nil {
		return &Result{similar: similar}, nil
	}

	return nil, ErrStubNotFound
}

// processStubsParallel processes stubs in parallel using goroutines.
func (s *searcher) processStubsParallel(query Query, stubs []*Stub) (*Result, error) {
	const chunkSize = 50

	numChunks := (len(stubs) + chunkSize - 1) / chunkSize
	results := make(chan chunkOutcome, numChunks)

	for i := range numChunks {
		start := i * chunkSize

		end := min(start+chunkSize, len(stubs))
		go func(chunkStubs []*Stub) {
			results <- s.processChunk(query, chunkStubs)
		}(stubs[start:end])
	}

	bestMatches, mostSimilar := collectChunkResults(results, numChunks)
	sortRankedMatches(bestMatches)

	return s.resultFromRankedAndSimilar(query, bestMatches, mostSimilar.stub)
}

// processChunk runs the sequential candidate collection over one chunk, so the
// parallel path uses identical match/similar criteria as the sequential path.
func (s *searcher) processChunk(query Query, chunkStubs []*Stub) chunkOutcome {
	matches, mostSimilar := s.collectSequentialCandidates(query, chunkStubs)

	return chunkOutcome{matches: matches, mostSimilar: mostSimilar}
}

func collectChunkResults(results chan chunkOutcome, numChunks int) ([]rankedMatch, similarCandidate) {
	bestMatches := make([]rankedMatch, 0, numChunks)

	var bestSimilar similarCandidate

	for range numChunks {
		result := <-results
		bestMatches = append(bestMatches, result.matches...)

		if betterSimilar(bestSimilar, result.mostSimilar) {
			bestSimilar = result.mostSimilar
		}
	}

	return bestMatches, bestSimilar
}
