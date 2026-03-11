package stuber

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeQueryFinder struct {
	result *Result
	err    error
	called bool
	query  Query
}

func (f *fakeQueryFinder) Find(query Query) (*Result, error) {
	f.called = true
	f.query = query

	return f.result, f.err
}

type fakeTraceStageBuilder struct {
	called bool
	query  Query
}

func (f *fakeTraceStageBuilder) addLookupStages(query Query, _ traceCollector) {
	f.called = true
	f.query = query
}

type fakeTraceCollector struct {
	matchedID *uuid.UUID
}

func (f *fakeTraceCollector) addStage(_ string, _, _ int) {}

func (f *fakeTraceCollector) setFallbackToMethod(_ bool) {}

func (f *fakeTraceCollector) setMatchedStubID(id *uuid.UUID) {
	f.matchedID = id
}

func TestSearchWithTraceDecoratorWithoutTrace(t *testing.T) {
	t.Parallel()

	finder := &fakeQueryFinder{}
	stageBuilder := &fakeTraceStageBuilder{}

	decorator := newSearchWithTraceDecorator(finder, stageBuilder, nil)
	_, _ = decorator.Find(Query{Service: "svc", Method: "M"})

	require.True(t, finder.called)
	require.False(t, stageBuilder.called)
}

func TestSearchWithTraceDecoratorAddsStagesAndMatchedID(t *testing.T) {
	t.Parallel()

	found := &Stub{ID: uuid.New()}
	finder := &fakeQueryFinder{result: &Result{found: found}}
	stageBuilder := &fakeTraceStageBuilder{}
	collector := &fakeTraceCollector{}

	decorator := newSearchWithTraceDecorator(finder, stageBuilder, collector)
	result, err := decorator.Find(Query{Service: "svc", Method: "M"})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, finder.called)
	require.True(t, stageBuilder.called)
	require.NotNil(t, collector.matchedID)
	require.Equal(t, found.ID, *collector.matchedID)
}

func TestSearchWithTraceDecoratorReturnsBaseError(t *testing.T) {
	t.Parallel()

	expected := ErrStubNotFound
	finder := &fakeQueryFinder{err: expected}
	stageBuilder := &fakeTraceStageBuilder{}
	collector := &fakeTraceCollector{}

	decorator := newSearchWithTraceDecorator(finder, stageBuilder, collector)
	result, err := decorator.Find(Query{Service: "svc", Method: "M"})

	require.ErrorIs(t, err, expected)
	require.Nil(t, result)
	require.True(t, finder.called)
	require.True(t, stageBuilder.called)
	require.Nil(t, collector.matchedID)
}
