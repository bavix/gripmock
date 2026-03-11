package stuber

func addIDLookupStages(collector traceCollector, allCount int) {
	collector.addStage(traceStageID, allCount, 1)
	collector.addStage(traceStageSession, 1, 1)
	collector.addStage(traceStageTimes, 1, 1)
	collector.addStage(traceStageHeaders, 1, 1)
	collector.addStage(traceStageInput, 1, 1)
}

func (s *searcher) addRegularLookupStages(query Query, collector traceCollector, all []*Stub) {
	view := s.buildRegularLookupView(query, all)
	collector.addStage(traceStageServiceMethod, len(all), len(view.serviceMethod))
	collector.addStage(traceStageSession, len(view.serviceMethod), len(view.sessionFiltered))
	collector.addStage(traceStageTimes, len(view.sessionFiltered), len(view.timesFiltered))

	headersCount, inputCount := countHeadersAndInputMatches(s, query, view.timesFiltered)
	collector.addStage(traceStageHeaders, len(view.timesFiltered), headersCount)
	collector.addStage(traceStageInput, len(view.timesFiltered), inputCount)

	if view.hasFallback() {
		collector.setFallbackToMethod(true)
		collector.addStage(traceStageFallbackMethod, len(view.timesFiltered), s.countFallbackMethodCandidates(query))
	}
}

func (s *searcher) countFallbackMethodCandidates(query Query) int {
	count := 0
	for range s.lookup(query.Session).LookupMethodAvailable(query.Method) {
		count++
	}

	return count
}

func countHeadersAndInputMatches(s *searcher, query Query, stubs []*Stub) (int, int) {
	headersCount := 0
	inputCount := 0

	for _, stub := range stubs {
		headersMatched := doesQueryMatchStubHeaders(query, stub)
		if !headersMatched {
			continue
		}

		headersCount++

		if s.matcher.Match(query, stub) {
			inputCount++
		}
	}

	return headersCount, inputCount
}
