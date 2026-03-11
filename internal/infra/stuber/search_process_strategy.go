package stuber

type processStubsStrategy interface {
	Process(query Query, stubs []*Stub) (*Result, error)
}

type defaultProcessStubsStrategy struct {
	searcher *searcher
}

func newDefaultProcessStubsStrategy(searcher *searcher) *defaultProcessStubsStrategy {
	return &defaultProcessStubsStrategy{searcher: searcher}
}

func (s *defaultProcessStubsStrategy) Process(query Query, stubs []*Stub) (*Result, error) {
	return s.searcher.processStubs(query, stubs)
}
