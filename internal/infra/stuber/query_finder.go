package stuber

// QueryFinder abstracts stub lookup by query.
type QueryFinder interface {
	FindByQuery(query Query) (*Result, error)
}
