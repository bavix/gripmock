package stuber

type searchTraceStageBuilder struct {
	searcher *searcher
	all      []*Stub
}

func newSearchTraceStageBuilder(searcher *searcher, all []*Stub) *searchTraceStageBuilder {
	return &searchTraceStageBuilder{searcher: searcher, all: all}
}

func (b *searchTraceStageBuilder) addLookupStages(query Query, collector traceCollector) {
	all := b.all

	if query.ID != nil {
		addIDLookupStages(collector, len(all))

		return
	}

	b.searcher.addRegularLookupStages(query, collector, all)
}
