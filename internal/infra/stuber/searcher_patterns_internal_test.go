package stuber

import (
	"iter"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type instrumentedLookupProvider struct {
	built        bool
	idCalls      int
	serviceCalls int
	methodCalls  int
}

func (p *instrumentedLookupProvider) build(s *searcher, session string) *searcherLookup {
	p.built = true

	return &searcherLookup{
		id:      &instrumentedIDLookup{searcher: s, provider: p},
		service: &instrumentedServiceLookup{searcher: s, provider: p, session: session},
		method:  &instrumentedMethodLookup{searcher: s, provider: p, session: session},
	}
}

type instrumentedIDLookup struct {
	searcher *searcher
	provider *instrumentedLookupProvider
}

func (l *instrumentedIDLookup) LookupID(id uuid.UUID) *Stub {
	l.provider.idCalls++

	return l.searcher.findByID(id)
}

type instrumentedServiceLookup struct {
	searcher *searcher
	provider *instrumentedLookupProvider
	session  string
}

func (l *instrumentedServiceLookup) LookupServiceAvailable(service, method string) (iter.Seq[*Stub], error) {
	l.provider.serviceCalls++

	seq, err := l.searcher.storage.findAllAvailable(service, method, l.session)
	if err != nil {
		return nil, err
	}

	return l.searcher.filterNotExhaustedSeq(seq, l.session), nil
}

type instrumentedMethodLookup struct {
	searcher *searcher
	provider *instrumentedLookupProvider
	session  string
}

func (l *instrumentedMethodLookup) HasMethodAvailable(method string) bool {
	return l.searcher.storage.hasMethodAvailable(method, l.session)
}

func (l *instrumentedMethodLookup) LookupMethodAvailable(method string) iter.Seq[*Stub] {
	l.provider.methodCalls++

	return l.searcher.filterNotExhaustedSeq(l.searcher.storage.findByMethodAvailable(method, l.session), l.session)
}

type inspectTestEnv struct {
	b        *Budgerigar
	provider *instrumentedLookupProvider
}

func newInspectTestEnv() inspectTestEnv {
	provider := &instrumentedLookupProvider{}
	s := newSearcherWithOptions(searcherOptions{lookupProvider: provider})

	return inspectTestEnv{
		b:        &Budgerigar{searcher: s},
		provider: provider,
	}
}

func (e *inspectTestEnv) addCandidate(service, method string) *Stub {
	candidate := &Stub{
		ID:      uuid.New(),
		Service: service,
		Method:  method,
		Input:   InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	}

	e.b.searcher.upsert(candidate)

	return candidate
}

func TestPatternCompositionFallbackPath(t *testing.T) {
	t.Parallel()

	env := newInspectTestEnv()
	candidate := env.addCandidate("other.service", "Hello")

	report := env.b.InspectQuery(Query{
		Service: "missing.service",
		Method:  "Hello",
		Input:   []map[string]any{{"name": "Alex"}},
	})

	require.True(t, env.provider.built)
	require.Positive(t, env.provider.serviceCalls)
	require.Positive(t, env.provider.methodCalls)
	require.Equal(t, 0, env.provider.idCalls)
	require.True(t, report.FallbackToMethod)
	require.NotNil(t, report.MatchedStubID)
	require.Equal(t, candidate.ID, *report.MatchedStubID)
}

func TestPatternCompositionIDPath(t *testing.T) {
	t.Parallel()

	env := newInspectTestEnv()
	candidate := env.addCandidate("svc", "Hello")

	report := env.b.InspectQuery(Query{
		ID:      &candidate.ID,
		Service: "svc",
		Method:  "Hello",
		Input:   []map[string]any{{"name": "Alex"}},
	})

	require.True(t, env.provider.built)
	require.Positive(t, env.provider.idCalls)
	require.NotNil(t, report.MatchedStubID)
	require.Equal(t, candidate.ID, *report.MatchedStubID)
}
