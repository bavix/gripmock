package stuber

// searchWithTraceDecorator is a thin decorator over searcher that
// enables trace collection for inspect/debug use-cases.
//
// It keeps core search behavior in searcher unchanged while isolating
// trace wiring from business endpoints.
type searchWithTraceDecorator struct {
	base   queryFinder
	stages traceStageBuilder
	trace  traceCollector
}

type traceStageBuilder interface {
	addLookupStages(query Query, collector traceCollector)
}

func newSearchWithTraceDecorator(base queryFinder, stages traceStageBuilder, trace traceCollector) *searchWithTraceDecorator {
	return &searchWithTraceDecorator{base: base, stages: stages, trace: trace}
}

func (d *searchWithTraceDecorator) Find(query Query) (*Result, error) {
	result, err := d.base.Find(query)
	if d.trace == nil {
		return result, err
	}

	if d.stages != nil {
		d.stages.addLookupStages(query, d.trace)
	}

	if result != nil && result.Found() != nil {
		id := result.Found().ID
		d.trace.setMatchedStubID(&id)
	}

	return result, err
}
