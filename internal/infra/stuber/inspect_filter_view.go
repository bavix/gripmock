package stuber

type regularLookupView struct {
	serviceMethod   []*Stub
	sessionFiltered []*Stub
	timesFiltered   []*Stub
}

func (s *searcher) buildRegularLookupView(query Query, all []*Stub) regularLookupView {
	serviceMethod := filterByServiceMethod(all, query.Service, query.Method)

	sessionFiltered := filterBySession(serviceMethod, query.Session)
	timesFiltered := s.filterExhaustedStubs(sessionFiltered, query.Session)

	return regularLookupView{
		serviceMethod:   serviceMethod,
		sessionFiltered: sessionFiltered,
		timesFiltered:   timesFiltered,
	}
}

func (v regularLookupView) hasFallback() bool {
	return len(v.serviceMethod) == 0
}
