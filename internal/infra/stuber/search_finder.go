package stuber

type queryFinder interface {
	Find(query Query) (*Result, error)
}

func (s *searcher) Find(query Query) (*Result, error) {
	return s.find(query)
}
