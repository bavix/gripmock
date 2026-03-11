package stuber

type searchTraceFinderFactory struct {
	base   queryFinder
	stages traceStageBuilder
}

func newSearchTraceFinderFactory(base queryFinder, stages traceStageBuilder) *searchTraceFinderFactory {
	return &searchTraceFinderFactory{base: base, stages: stages}
}

func (f *searchTraceFinderFactory) New(trace traceCollector) *searchWithTraceDecorator {
	return newSearchWithTraceDecorator(f.base, f.stages, trace)
}
