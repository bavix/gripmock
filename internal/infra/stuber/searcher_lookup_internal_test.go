package stuber

import (
	"iter"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeIDLookup struct {
	called bool
	stub   *Stub
}

func (f *fakeIDLookup) LookupID(_ uuid.UUID) *Stub {
	f.called = true

	return f.stub
}

type fakeServiceLookup struct {
	called bool
	err    error
	stubs  []*Stub
}

func (f *fakeServiceLookup) LookupServiceAvailable(_, _ string) (iter.Seq[*Stub], error) {
	f.called = true

	return seqFromStubs(f.stubs), f.err
}

type fakeMethodLookup struct {
	called bool
	stubs  []*Stub
}

func (f *fakeMethodLookup) HasMethodAvailable(_ string) bool {
	return len(f.stubs) > 0
}

func (f *fakeMethodLookup) LookupMethodAvailable(_ string) iter.Seq[*Stub] {
	f.called = true

	return seqFromStubs(f.stubs)
}

type fakeLookupProvider struct {
	called bool
	lookup *searcherLookup
}

func (f *fakeLookupProvider) build(_ *searcher, _ string) *searcherLookup {
	f.called = true

	return f.lookup
}

func seqFromStubs(stubs []*Stub) iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		for _, stub := range stubs {
			if !yield(stub) {
				return
			}
		}
	}
}

func TestSearcherLookupFactoryMethodFallback(t *testing.T) {
	t.Parallel()

	candidate := &Stub{
		ID:      uuid.New(),
		Service: "other.service",
		Method:  "Hello",
		Input:   InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	}

	id := &fakeIDLookup{}
	service := &fakeServiceLookup{err: ErrServiceNotFound}
	method := &fakeMethodLookup{stubs: []*Stub{candidate}}

	s := newSearcherWithOptions(searcherOptions{
		lookupFactory: searcherLookupFactory{
			newID: func(_ *searcher) idLookup {
				return id
			},
			newService: func(_ *searcher, _ string) serviceLookup {
				return service
			},
			newMethod: func(_ *searcher, _ string) methodLookup {
				return method
			},
		},
	})

	result, err := s.searchOptimized(Query{
		Service: "missing.service",
		Method:  "Hello",
		Input:   []map[string]any{{"name": "Alex"}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())
	require.Equal(t, candidate.ID, result.Found().ID)
	require.True(t, service.called)
	require.True(t, method.called)
	require.False(t, id.called)
}

func TestSearcherLookupFactoryIDPath(t *testing.T) {
	t.Parallel()

	stub := &Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Input:   InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	}

	id := &fakeIDLookup{stub: stub}

	s := newSearcherWithOptions(searcherOptions{
		lookupFactory: searcherLookupFactory{
			newID: func(_ *searcher) idLookup {
				return id
			},
			newService: func(_ *searcher, _ string) serviceLookup {
				return &fakeServiceLookup{}
			},
			newMethod: func(_ *searcher, _ string) methodLookup {
				return &fakeMethodLookup{}
			},
		},
	})

	s.upsert(stub)

	result, err := s.searchByID(Query{ID: &stub.ID, Service: "svc", Method: "M"})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())
	require.Equal(t, stub.ID, result.Found().ID)
	require.True(t, id.called)
}

func TestSearcherLookupProviderHasPriorityOverFactory(t *testing.T) {
	t.Parallel()

	candidate := &Stub{
		ID:      uuid.New(),
		Service: "other.service",
		Method:  "Hello",
		Input:   InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	}

	providerID := &fakeIDLookup{}
	providerService := &fakeServiceLookup{err: ErrServiceNotFound}
	providerMethod := &fakeMethodLookup{stubs: []*Stub{candidate}}

	provider := &fakeLookupProvider{lookup: &searcherLookup{
		id:      providerID,
		service: providerService,
		method:  providerMethod,
	}}

	s := newSearcherWithOptions(searcherOptions{
		lookupProvider: provider,
		lookupFactory: searcherLookupFactory{
			newID: func(_ *searcher) idLookup {
				panic("factory id lookup should not be used when provider exists")
			},
			newService: func(_ *searcher, _ string) serviceLookup {
				panic("factory service lookup should not be used when provider exists")
			},
			newMethod: func(_ *searcher, _ string) methodLookup {
				panic("factory method lookup should not be used when provider exists")
			},
		},
	})

	result, err := s.searchOptimized(Query{
		Service: "missing.service",
		Method:  "Hello",
		Input:   []map[string]any{{"name": "Alex"}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())
	require.Equal(t, candidate.ID, result.Found().ID)
	require.True(t, provider.called)
	require.True(t, providerService.called)
	require.True(t, providerMethod.called)
	require.False(t, providerID.called)
}

func TestSearcherLookupFactoryUsedWhenProviderMissing(t *testing.T) {
	t.Parallel()

	candidate := &Stub{
		ID:      uuid.New(),
		Service: "other.service",
		Method:  "Hello",
		Input:   InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	}

	factoryService := &fakeServiceLookup{err: ErrServiceNotFound}
	factoryMethod := &fakeMethodLookup{stubs: []*Stub{candidate}}

	factoryConstructed := false

	s := newSearcherWithOptions(searcherOptions{
		lookupFactory: searcherLookupFactory{
			newID: func(_ *searcher) idLookup {
				factoryConstructed = true

				return &fakeIDLookup{}
			},
			newService: func(_ *searcher, _ string) serviceLookup {
				factoryConstructed = true

				return factoryService
			},
			newMethod: func(_ *searcher, _ string) methodLookup {
				factoryConstructed = true

				return factoryMethod
			},
		},
	})

	result, err := s.searchOptimized(Query{
		Service: "missing.service",
		Method:  "Hello",
		Input:   []map[string]any{{"name": "Alex"}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())
	require.Equal(t, candidate.ID, result.Found().ID)
	require.True(t, factoryConstructed)
	require.True(t, factoryService.called)
	require.True(t, factoryMethod.called)
}

func TestSessionLookupFallsBackToGlobalWhenSessionEmpty(t *testing.T) {
	t.Parallel()

	global := &Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Input:   InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  Output{Data: map[string]any{"scope": "global"}},
	}

	s := newSearcher()
	s.upsert(global)

	result, err := s.find(Query{
		Service: "svc",
		Method:  "M",
		Session: "S1",
		Input:   []map[string]any{{"name": "Alex"}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())
	require.Equal(t, global.ID, result.Found().ID)
}

func TestSessionLookupMergesSessionAndGlobalStorage(t *testing.T) {
	t.Parallel()

	global := &Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Input:   InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  Output{Data: map[string]any{"scope": "global"}},
	}

	session := &Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Session: "S1",
		Input:   InputData{Equals: map[string]any{"name": "Bob"}},
		Output:  Output{Data: map[string]any{"scope": "session"}},
	}

	s := newSearcher()
	s.upsert(global, session)

	result, err := s.find(Query{
		Service: "svc",
		Method:  "M",
		Session: "S1",
		Input:   []map[string]any{{"name": "Alex"}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())
	require.Equal(t, global.ID, result.Found().ID)

	result, err = s.find(Query{
		Service: "svc",
		Method:  "M",
		Session: "S1",
		Input:   []map[string]any{{"name": "Bob"}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())
	require.Equal(t, session.ID, result.Found().ID)
}
