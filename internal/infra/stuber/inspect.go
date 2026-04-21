package stuber

import (
	"github.com/google/uuid"

	"github.com/bavix/features"
)

type searchTrace struct {
	stages           []InspectStage
	fallbackToMethod bool
	candidates       []InspectCandidate
	matchedStubID    *uuid.UUID
}

const inspectStagesCap = 8

// searchTrace is request-scoped and mutated only on the caller goroutine.
// Parallel search workers do not write into trace directly, so extra mutex
// synchronization is not required here.

func newSearchTrace() *searchTrace {
	return &searchTrace{
		stages: make([]InspectStage, 0, inspectStagesCap),
	}
}

func (t *searchTrace) addStage(name string, before, after int) {
	removed := max(before-after, 0)

	t.stages = append(t.stages, InspectStage{
		Name:    name,
		Before:  before,
		After:   after,
		Removed: removed,
	})
}

func (t *searchTrace) setFallbackToMethod(v bool) {
	t.fallbackToMethod = v
}

func (t *searchTrace) initCandidates(candidates []InspectCandidate) {
	t.candidates = candidates
}

func (t *searchTrace) setMatchedStubID(id *uuid.UUID) {
	t.matchedStubID = id
}

func (t *searchTrace) finalizeCandidates() []InspectCandidate {
	for i := range t.candidates {
		candidate := &t.candidates[i]
		candidate.Matched = t.matchedStubID != nil && candidate.ID == *t.matchedStubID

		for j := range candidate.Events {
			if candidate.Events[j].Stage != traceStageSelected {
				continue
			}

			if candidate.Matched {
				candidate.Events[j].Result = traceResultPassed
				candidate.Events[j].Reason = ""
			}

			break
		}
	}

	return t.candidates
}

type InspectStage struct {
	Name    string `json:"name"`
	Before  int    `json:"before"`
	After   int    `json:"after"`
	Removed int    `json:"removed"`
}

type InspectCandidate struct {
	ID               uuid.UUID               `json:"id"`
	Service          string                  `json:"service"`
	Method           string                  `json:"method"`
	Session          string                  `json:"session,omitempty"`
	Priority         int                     `json:"priority"`
	Times            int                     `json:"times"`
	Used             int                     `json:"used"`
	VisibleBySession bool                    `json:"visibleBySession"`
	WithinTimes      bool                    `json:"withinTimes"`
	HeadersMatched   bool                    `json:"headersMatched"`
	InputMatched     bool                    `json:"inputMatched"`
	Matched          bool                    `json:"matched"`
	Specificity      int                     `json:"specificity"`
	Score            float64                 `json:"score"`
	ExcludedBy       []string                `json:"excludedBy,omitempty"`
	Events           []InspectCandidateEvent `json:"events,omitempty"`
}

type InspectCandidateEvent struct {
	Stage  string `json:"stage"`
	Result string `json:"result"`
	Reason string `json:"reason,omitempty"`
}

type InspectReport struct {
	Service          string             `json:"service"`
	Method           string             `json:"method"`
	Session          string             `json:"session,omitempty"`
	FallbackToMethod bool               `json:"fallbackToMethod"`
	Error            *string            `json:"error,omitempty"`
	MatchedStubID    *uuid.UUID         `json:"matchedStubId,omitempty"`
	SimilarStubID    *uuid.UUID         `json:"similarStubId,omitempty"`
	Stages           []InspectStage     `json:"stages"`
	Candidates       []InspectCandidate `json:"candidates"`
}

func (b *Budgerigar) InspectQuery(query Query) InspectReport {
	return b.searcher.inspect(query)
}

func (s *searcher) inspect(query Query) InspectReport {
	query.toggles = features.New(RequestInternalFlag)

	trace := newSearchTrace()
	all := s.all()
	fallbackToMethod := s.detectFallbackToMethod(query)

	trace.initCandidates(s.collectTraceCandidates(query, all, fallbackToMethod))

	factory := newSearchTraceFinderFactory(s, newSearchTraceStageBuilder(s, all))
	finder := factory.New(trace)

	result, err := finder.Find(query)
	if err != nil {
		result = nil
	}

	var matchedID *uuid.UUID

	if result != nil && result.Found() != nil {
		id := result.Found().ID
		matchedID = &id
	}

	trace.setMatchedStubID(matchedID)

	report := InspectReport{
		Service:    query.Service,
		Method:     query.Method,
		Session:    query.Session,
		Stages:     trace.stages,
		Candidates: trace.finalizeCandidates(),
	}
	report.FallbackToMethod = trace.fallbackToMethod

	if err != nil {
		msg := err.Error()
		report.Error = &msg
	}

	applyResultIDs(&report, result)

	return report
}

func (s *searcher) detectFallbackToMethod(query Query) bool {
	if query.ID != nil {
		return false
	}

	lookup := s.lookup(query.Session)
	if _, err := lookup.LookupServiceAvailable(query.Service, query.Method); err != nil && lookup.HasMethodAvailable(query.Method) {
		return true
	}

	return false
}

func applyResultIDs(report *InspectReport, result *Result) {
	if result == nil {
		return
	}

	if found := result.Found(); found != nil {
		id := found.ID
		report.MatchedStubID = &id
	}

	if similar := result.Similar(); similar != nil {
		id := similar.ID
		report.SimilarStubID = &id
	}
}
