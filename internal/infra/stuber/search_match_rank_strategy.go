package stuber

type matchStrategy interface {
	Match(query Query, stub *Stub) bool
}

type rankStrategy interface {
	Score(query Query, stub *Stub) float64
	Specificity(query Query, stub *Stub) int
	FieldCount(stub *Stub) int
}

type defaultMatchStrategy struct {
	searcher *searcher
}

type defaultRankStrategy struct {
	searcher *searcher
}

func newDefaultMatchStrategy(searcher *searcher) *defaultMatchStrategy {
	return &defaultMatchStrategy{searcher: searcher}
}

func (d *defaultMatchStrategy) Match(query Query, stub *Stub) bool {
	return d.searcher.fastMatchV2(query, stub)
}

func newDefaultRankStrategy(searcher *searcher) *defaultRankStrategy {
	return &defaultRankStrategy{searcher: searcher}
}

func (d *defaultRankStrategy) Score(query Query, stub *Stub) float64 {
	return d.searcher.fastRankV2(query, stub)
}

func (d *defaultRankStrategy) Specificity(query Query, stub *Stub) int {
	return d.searcher.calcSpecificity(stub, query)
}

func (d *defaultRankStrategy) FieldCount(stub *Stub) int {
	return countStubFields(stub)
}

func countStubFields(stub *Stub) int {
	count := len(stub.Input.Equals) + len(stub.Input.Contains) + len(stub.Input.Matches)
	count += len(stub.Headers.Equals) + len(stub.Headers.Contains) + len(stub.Headers.Matches)

	for _, input := range stub.Inputs {
		count += len(input.Equals) + len(input.Contains) + len(input.Matches)
		count += countAnyOfFields(input.AnyOf)
	}

	count += countAnyOfFields(stub.Input.AnyOf)

	for _, alt := range stub.Headers.AnyOf {
		count += len(alt.Equals) + len(alt.Contains) + len(alt.Matches)
	}

	return count
}

func countAnyOfFields(anyOf []AnyOfElement) int {
	var n int

	for _, alt := range anyOf {
		n += len(alt.Equals) + len(alt.Contains) + len(alt.Matches)
	}

	return n
}
