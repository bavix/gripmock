package stuber

const (
	inspectEventCount        = 6
	inspectExcludedReasonCap = 4
)

func (s *searcher) collectTraceCandidates(query Query, stubs []*Stub, fallbackToMethod bool) []InspectCandidate {
	out := make([]InspectCandidate, len(stubs))
	eventsPool := make([]InspectCandidateEvent, len(stubs)*inspectEventCount)
	reasonsPool := make([]string, len(stubs)*inspectExcludedReasonCap)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for i, stub := range stubs {
		events := eventsPool[i*inspectEventCount : (i+1)*inspectEventCount]
		reasons := reasonsPool[i*inspectExcludedReasonCap : (i+1)*inspectExcludedReasonCap]
		out[i] = s.buildTraceCandidate(query, stub, fallbackToMethod, events, reasons)
	}

	return out
}

type traceEval struct {
	used           int
	times          int
	score          float64
	specificity    int
	withinTimes    bool
	visible        bool
	headersMatched bool
	inputMatched   bool
	excludedBy     []string
	events         []InspectCandidateEvent
}

func (s *searcher) buildTraceCandidate(
	query Query,
	stub *Stub,
	fallbackToMethod bool,
	events []InspectCandidateEvent,
	reasons []string,
) InspectCandidate {
	eval := s.evalTraceCandidate(query, stub, fallbackToMethod, events, reasons)

	return InspectCandidate{
		ID:               stub.ID,
		Service:          stub.Service,
		Method:           stub.Method,
		Session:          stub.Session,
		Priority:         stub.Priority,
		Times:            eval.times,
		Used:             eval.used,
		VisibleBySession: eval.visible,
		WithinTimes:      eval.withinTimes,
		HeadersMatched:   eval.headersMatched,
		InputMatched:     eval.inputMatched,
		Matched:          false,
		Specificity:      eval.specificity,
		Score:            eval.score,
		ExcludedBy:       eval.excludedBy,
		Events:           eval.events,
	}
}

func (s *searcher) evalTraceCandidate(
	query Query,
	stub *Stub,
	fallbackToMethod bool,
	events []InspectCandidateEvent,
	reasons []string,
) traceEval {
	used := s.stubCallCount[callCountKey{id: stub.ID, session: query.Session}]
	times := stub.EffectiveTimes()
	withinTimes := times <= 0 || used < times
	visible := isStubVisibleForSession(stub.Session, query.Session)
	headersMatched := doesQueryMatchStubHeaders(query, stub)
	inputMatched := s.fastMatchV2(query, stub)
	ranked := s.rankedMatchFor(query, stub)

	routeStage, routePassed, routeReason, reasonCount := evalRoute(query, stub, fallbackToMethod, reasons)
	reasonCount = appendReasonToBuffer(reasons, reasonCount, !visible, traceReasonSession)
	reasonCount = appendReasonToBuffer(reasons, reasonCount, !withinTimes, traceReasonTimes)
	reasonCount = appendReasonToBuffer(reasons, reasonCount, query.ID == nil && !headersMatched, traceReasonHeaders)
	reasonCount = appendReasonToBuffer(reasons, reasonCount, query.ID == nil && !inputMatched, traceReasonInput)

	buildTraceEvents(events, query, routeStage, routePassed, routeReason, visible, withinTimes, headersMatched, inputMatched)

	return traceEval{
		used:           used,
		times:          times,
		score:          ranked.totalScore,
		specificity:    ranked.specificity,
		withinTimes:    withinTimes,
		visible:        visible,
		headersMatched: headersMatched,
		inputMatched:   inputMatched,
		excludedBy:     reasons[:reasonCount],
		events:         events,
	}
}

func evalRoute(query Query, stub *Stub, fallbackToMethod bool, reasons []string) (string, bool, string, int) {
	routeStage := traceStageServiceMethod
	routePassed := stub.Service == query.Service && stub.Method == query.Method
	reasonCount := 0

	if query.ID != nil {
		routeStage = traceStageID
		routePassed = stub.ID == *query.ID

		routeReason := reasonIf(!routePassed, traceReasonID)
		reasonCount = appendReasonToBuffer(reasons, reasonCount, !routePassed, traceReasonID)

		return routeStage, routePassed, routeReason, reasonCount
	}

	if fallbackToMethod {
		routeStage = traceStageFallbackMethod
		routePassed = stub.Method == query.Method

		routeReason := reasonIf(!routePassed, traceReasonMethod)
		reasonCount = appendReasonToBuffer(reasons, reasonCount, !routePassed, traceReasonMethod)

		return routeStage, routePassed, routeReason, reasonCount
	}

	serviceMismatch := stub.Service != query.Service
	methodMismatch := stub.Method != query.Method

	reasonCount = appendReasonToBuffer(reasons, reasonCount, serviceMismatch, traceReasonService)
	reasonCount = appendReasonToBuffer(reasons, reasonCount, methodMismatch, traceReasonMethod)

	routeReason := reasonIf(serviceMismatch, traceReasonService)
	if routeReason == "" {
		routeReason = reasonIf(methodMismatch, traceReasonMethod)
	}

	return routeStage, routePassed, routeReason, reasonCount
}

func appendReasonToBuffer(reasons []string, pos int, condition bool, reason string) int {
	if !condition {
		return pos
	}

	reasons[pos] = reason

	return pos + 1
}

func buildTraceEvents(
	events []InspectCandidateEvent,
	query Query,
	routeStage string,
	routePassed bool,
	routeReason string,
	visible bool,
	withinTimes bool,
	headersMatched bool,
	inputMatched bool,
) {
	events[0] = InspectCandidateEvent{Stage: routeStage, Result: boolResult(routePassed), Reason: routeReason}
	events[1] = InspectCandidateEvent{Stage: traceStageSession, Result: boolResult(visible), Reason: reasonIf(!visible, traceReasonSession)}
	events[2] = InspectCandidateEvent{
		Stage:  traceStageTimes,
		Result: boolResult(withinTimes),
		Reason: reasonIf(!withinTimes, traceReasonTimes),
	}

	if query.ID != nil {
		events[3] = InspectCandidateEvent{Stage: traceStageHeaders, Result: traceResultSkipped, Reason: traceReasonIDLookup}
		events[4] = InspectCandidateEvent{Stage: traceStageInput, Result: traceResultSkipped, Reason: traceReasonIDLookup}
	} else {
		events[3] = InspectCandidateEvent{
			Stage:  traceStageHeaders,
			Result: boolResult(headersMatched),
			Reason: reasonIf(!headersMatched, traceReasonHeaders),
		}
		events[4] = InspectCandidateEvent{
			Stage:  traceStageInput,
			Result: boolResult(inputMatched),
			Reason: reasonIf(!inputMatched, traceReasonInput),
		}
	}

	events[5] = InspectCandidateEvent{Stage: traceStageSelected, Result: traceResultFailed, Reason: traceReasonNotSelect}
}
