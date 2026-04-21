package stuber_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type inspectConsistencyCase struct {
	name             string
	stubs            []*stuber.Stub
	query            stuber.Query
	beforeInspect    func(s *stuber.Budgerigar)
	expectedMatched  uuid.UUID
	expectedFallback bool
}

//nolint:funlen
func TestInspectConsistencyWithFindByQuery(t *testing.T) {
	t.Parallel()

	newStub := func(service, method, marker string) *stuber.Stub {
		return &stuber.Stub{
			ID:      uuid.New(),
			Service: service,
			Method:  method,
			Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
			Output:  stuber.Output{Data: map[string]any{"marker": marker}},
		}
	}

	t.Run("exactPreferredOverContains", func(t *testing.T) {
		t.Parallel()

		exact := newStub("s.demo", "Hello", "exact")
		contains := newStub("s.demo", "Hello", "contains")
		contains.Input = stuber.InputData{Contains: map[string]any{"name": "Al"}}

		tc := inspectConsistencyCase{
			name:            "exactPreferredOverContains",
			stubs:           []*stuber.Stub{exact, contains},
			query:           stuber.Query{Service: "s.demo", Method: "Hello", Input: []map[string]any{{"name": "Alex"}}, Headers: nil},
			expectedMatched: exact.ID,
		}

		runInspectConsistencyCase(t, tc)
	})

	t.Run("headersAffectSelection", func(t *testing.T) {
		t.Parallel()

		prod := newStub("s.demo", "Hello", "prod")
		prod.Headers = stuber.InputHeader{Equals: map[string]any{"x-env": "prod"}}
		generic := newStub("s.demo", "Hello", "generic")

		tc := inspectConsistencyCase{
			name:  "headersAffectSelection",
			stubs: []*stuber.Stub{prod, generic},
			query: stuber.Query{
				Service: "s.demo",
				Method:  "Hello",
				Headers: map[string]any{"x-env": "prod"},
				Input:   []map[string]any{{"name": "Alex"}},
				Session: "",
			},
			expectedMatched: prod.ID,
		}

		runInspectConsistencyCase(t, tc)
	})

	t.Run("sessionScopedBeatsGlobal", func(t *testing.T) {
		t.Parallel()

		sessionStub := newStub("s.demo", "Hello", "session")
		sessionStub.Session = "s1"
		globalStub := newStub("s.demo", "Hello", "global")
		globalStub.Input = stuber.InputData{Contains: map[string]any{"name": "Al"}}

		tc := inspectConsistencyCase{
			name:  "sessionScopedBeatsGlobal",
			stubs: []*stuber.Stub{sessionStub, globalStub},
			query: stuber.Query{
				Service: "s.demo",
				Method:  "Hello",
				Session: "s1",
				Input:   []map[string]any{{"name": "Alex"}},
				Headers: nil,
			},
			expectedMatched: sessionStub.ID,
		}

		runInspectConsistencyCase(t, tc)
	})

	t.Run("timesExhaustionMatchesFallback", func(t *testing.T) {
		t.Parallel()

		once := newStub("s.demo", "Hello", "once")
		once.Options = stuber.StubOptions{Times: 1}
		once.Priority = 10
		fallback := newStub("s.demo", "Hello", "fallback")

		tc := inspectConsistencyCase{
			name:            "timesExhaustionMatchesFallback",
			stubs:           []*stuber.Stub{once, fallback},
			query:           stuber.Query{Service: "s.demo", Method: "Hello", Input: []map[string]any{{"name": "Alex"}}},
			expectedMatched: fallback.ID,
			beforeInspect: func(s *stuber.Budgerigar) {
				_, err := s.FindByQuery(stuber.Query{Service: "s.demo", Method: "Hello", Input: []map[string]any{{"name": "Alex"}}})
				require.NoError(t, err)
			},
		}

		runInspectConsistencyCase(t, tc)
	})

	t.Run("fallbackToMethodWhenServiceHasNoStubs", func(t *testing.T) {
		t.Parallel()

		methodOnly := newStub("other.service", "Hello", "methodonly")

		tc := inspectConsistencyCase{
			name:             "fallbackToMethodWhenServiceHasNoStubs",
			stubs:            []*stuber.Stub{methodOnly},
			query:            stuber.Query{Service: "s.demo", Method: "Hello", Input: []map[string]any{{"name": "Alex"}}},
			expectedMatched:  methodOnly.ID,
			expectedFallback: true,
		}

		runInspectConsistencyCase(t, tc)
	})
}

func runInspectConsistencyCase(t *testing.T, tc inspectConsistencyCase) {
	t.Helper()

	s := stuber.NewBudgerigar()
	s.PutMany(tc.stubs...)

	if tc.beforeInspect != nil {
		tc.beforeInspect(s)
	}

	findResult, err := s.FindByQuery(tc.query)
	require.NoError(t, err, tc.name)
	require.NotNil(t, findResult)
	require.NotNil(t, findResult.Found())
	require.Equal(t, tc.expectedMatched, findResult.Found().ID)

	report := s.InspectQuery(tc.query)
	require.NotNil(t, report.MatchedStubID)
	require.Equal(t, tc.expectedMatched, *report.MatchedStubID)
	require.Equal(t, tc.expectedFallback, report.FallbackToMethod)
	require.NotEmpty(t, report.Stages)
	require.NotEmpty(t, report.Candidates)

	matchedCount := 0

	for _, candidate := range report.Candidates {
		for _, reason := range candidate.ExcludedBy {
			require.NotEqual(t, "route", reason)
		}

		require.NotEmpty(t, candidate.Events)

		hasSelected := false

		for _, event := range candidate.Events {
			if event.Stage == "selected" {
				hasSelected = true
			}
		}

		require.True(t, hasSelected)

		if candidate.Matched {
			matchedCount++

			require.Equal(t, tc.expectedMatched, candidate.ID)
		}
	}

	require.Equal(t, 1, matchedCount)

	requireStagesPresent(t, &report, "session", "times", "headers", "input")

	if findResult.Similar() != nil {
		require.NotNil(t, report.SimilarStubID)
		require.Equal(t, findResult.Similar().ID, *report.SimilarStubID)
	}
}

func requireStagesPresent(t *testing.T, report *stuber.InspectReport, expected ...string) {
	t.Helper()

	stageNames := make(map[string]struct{}, len(report.Stages))
	for _, stage := range report.Stages {
		stageNames[stage.Name] = struct{}{}
	}

	for _, name := range expected {
		_, ok := stageNames[name]
		require.True(t, ok, "expected stage %q to be present", name)
	}
}

//nolint:cyclop,gocognit,funlen
func TestInspectTraceStagesEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("fallbackMethodStagePresentAndNoServiceExclusion", func(t *testing.T) {
		t.Parallel()

		s := stuber.NewBudgerigar()
		candidate := &stuber.Stub{
			ID:      uuid.New(),
			Service: "other.service",
			Method:  "Hello",
			Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
			Output:  stuber.Output{Data: map[string]any{"ok": true}},
		}
		s.PutMany(candidate)

		report := s.InspectQuery(stuber.Query{
			Service: "missing.service",
			Method:  "Hello",
			Input:   []map[string]any{{"name": "Alex"}},
		})

		require.True(t, report.FallbackToMethod)
		require.NotEmpty(t, report.Stages)

		hasFallbackStage := false

		for _, stage := range report.Stages {
			if stage.Name == "fallback_method" {
				hasFallbackStage = true

				break
			}
		}

		require.True(t, hasFallbackStage)

		for _, c := range report.Candidates {
			if c.ID != candidate.ID {
				continue
			}

			for _, reason := range c.ExcludedBy {
				if reason == "service" {
					t.Fatal("service should not be exclusion reason in fallback-to-method mode")
				}
			}

			return
		}

		t.Fatal("fallback candidate not found")
	})

	t.Run("idLookupStagesPresent", func(t *testing.T) {
		t.Parallel()

		s := stuber.NewBudgerigar()
		stub := &stuber.Stub{
			ID:      uuid.New(),
			Service: "s.demo",
			Method:  "Hello",
			Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
			Output:  stuber.Output{Data: map[string]any{"ok": true}},
		}
		s.PutMany(stub)

		report := s.InspectQuery(stuber.Query{
			ID:      &stub.ID,
			Service: "s.demo",
			Method:  "Hello",
			Input:   []map[string]any{{"name": "Alex"}},
		})

		require.NotNil(t, report.MatchedStubID)
		require.Equal(t, stub.ID, *report.MatchedStubID)

		requireStagesPresent(t, &report, "id", "session", "times", "headers", "input")
	})

	t.Run("idLookupDoesNotUseInputOrHeadersAsExclusion", func(t *testing.T) {
		t.Parallel()

		s := stuber.NewBudgerigar()
		stub := &stuber.Stub{
			ID:      uuid.New(),
			Service: "s.demo",
			Method:  "Hello",
			Headers: stuber.InputHeader{Equals: map[string]any{"x-env": "prod"}},
			Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
			Output:  stuber.Output{Data: map[string]any{"ok": true}},
		}
		s.PutMany(stub)

		report := s.InspectQuery(stuber.Query{
			ID:      &stub.ID,
			Service: "s.demo",
			Method:  "Hello",
			Headers: map[string]any{"x-env": "stage"},
			Input:   []map[string]any{{"name": "not-alex"}},
		})

		require.NotNil(t, report.MatchedStubID)
		require.Equal(t, stub.ID, *report.MatchedStubID)

		for _, candidate := range report.Candidates {
			if candidate.ID != stub.ID {
				continue
			}

			hasSkippedHeaders := false
			hasSkippedInput := false

			for _, event := range candidate.Events {
				if event.Stage == "headers" && event.Result == "skipped" {
					hasSkippedHeaders = true
				}

				if event.Stage == "input" && event.Result == "skipped" {
					hasSkippedInput = true
				}
			}

			require.True(t, hasSkippedHeaders)
			require.True(t, hasSkippedInput)

			for _, reason := range candidate.ExcludedBy {
				if reason == "headers" || reason == "input" {
					t.Fatalf("unexpected exclusion reason %q for ID lookup", reason)
				}
			}

			return
		}

		t.Fatal("target candidate not found in inspect report")
	})
}

func TestInspectDoesNotConsumeTimes(t *testing.T) {
	t.Parallel()

	s := stuber.NewBudgerigar()
	oneShot := &stuber.Stub{
		ID:      uuid.New(),
		Service: "s.demo",
		Method:  "Hello",
		Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  stuber.Output{Data: map[string]any{"marker": "one"}},
		Options: stuber.StubOptions{Times: 1},
	}
	s.PutMany(oneShot)

	query := stuber.Query{Service: "s.demo", Method: "Hello", Input: []map[string]any{{"name": "Alex"}}}

	reportBefore := s.InspectQuery(query)
	require.NotNil(t, reportBefore.MatchedStubID)
	require.Equal(t, oneShot.ID, *reportBefore.MatchedStubID)

	first, err := s.FindByQuery(query)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotNil(t, first.Found())
	require.Equal(t, oneShot.ID, first.Found().ID)

	reportAfter := s.InspectQuery(query)
	require.Nil(t, reportAfter.MatchedStubID)

	_, err = s.FindByQuery(query)
	require.ErrorIs(t, err, stuber.ErrStubNotFound)
}

//nolint:funlen
func TestInspectCandidateEventFlagConsistency(t *testing.T) {
	t.Parallel()

	type inspectEventConsistencyCase struct {
		name         string
		stub         *stuber.Stub
		query        stuber.Query
		expectFailed map[string]struct{}
	}

	type eventAssertParams struct {
		candidate    *stuber.InspectCandidate
		eventByStage map[string]stuber.InspectCandidateEvent
		expectFailed map[string]struct{}
	}

	assertEventConsistency := func(t *testing.T, p eventAssertParams) {
		t.Helper()

		if _, ok := p.expectFailed["session"]; ok {
			require.False(t, p.candidate.VisibleBySession)

			event := p.eventByStage["session"]
			require.Equal(t, "failed", event.Result)
			require.Equal(t, "session", event.Reason)
		}

		if _, ok := p.expectFailed["headers"]; ok {
			require.False(t, p.candidate.HeadersMatched)

			event := p.eventByStage["headers"]
			require.Equal(t, "failed", event.Result)
			require.Equal(t, "headers", event.Reason)
		}

		if _, ok := p.expectFailed["input"]; ok {
			require.False(t, p.candidate.InputMatched)

			event := p.eventByStage["input"]
			require.Equal(t, "failed", event.Result)
			require.Equal(t, "input", event.Reason)
		}
	}

	findCandidateInReport := func(report *stuber.InspectReport, id uuid.UUID) *stuber.InspectCandidate {
		for i := range report.Candidates {
			if report.Candidates[i].ID == id {
				return &report.Candidates[i]
			}
		}

		return nil
	}

	eventByStageMap := func(candidate *stuber.InspectCandidate) map[string]stuber.InspectCandidateEvent {
		eventByStage := make(map[string]stuber.InspectCandidateEvent, len(candidate.Events))
		for _, event := range candidate.Events {
			eventByStage[event.Stage] = event
		}

		return eventByStage
	}

	testCases := []inspectEventConsistencyCase{
		{
			name: "sessionMismatch",
			stub: &stuber.Stub{
				ID:      uuid.New(),
				Service: "s.demo",
				Method:  "Hello",
				Session: "s1",
				Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
				Output:  stuber.Output{Data: map[string]any{"ok": true}},
			},
			query: stuber.Query{
				Service: "s.demo",
				Method:  "Hello",
				Session: "s2",
				Input:   []map[string]any{{"name": "Alex"}},
			},
			expectFailed: map[string]struct{}{"session": {}},
		},
		{
			name: "headersRequiredButMissing",
			stub: &stuber.Stub{
				ID:      uuid.New(),
				Service: "s.demo",
				Method:  "Hello",
				Headers: stuber.InputHeader{Equals: map[string]any{"x-env": "prod"}},
				Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
				Output:  stuber.Output{Data: map[string]any{"ok": true}},
			},
			query: stuber.Query{
				Service: "s.demo",
				Method:  "Hello",
				Input:   []map[string]any{{"name": "Alex"}},
			},
			expectFailed: map[string]struct{}{"headers": {}},
		},
		{
			name: "emptyInputArray",
			stub: &stuber.Stub{
				ID:      uuid.New(),
				Service: "s.demo",
				Method:  "Hello",
				Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
				Output:  stuber.Output{Data: map[string]any{"ok": true}},
			},
			query: stuber.Query{
				Service: "s.demo",
				Method:  "Hello",
				Input:   []map[string]any{},
			},
			expectFailed: map[string]struct{}{"input": {}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := stuber.NewBudgerigar()
			s.PutMany(tc.stub)

			report := s.InspectQuery(tc.query)
			require.NotEmpty(t, report.Candidates)

			candidate := findCandidateInReport(&report, tc.stub.ID)
			require.NotNil(t, candidate)
			require.NotEmpty(t, candidate.Events)

			assertEventConsistency(t, eventAssertParams{
				candidate:    candidate,
				eventByStage: eventByStageMap(candidate),
				expectFailed: tc.expectFailed,
			})
		})
	}
}
