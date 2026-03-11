package stuber

import (
	"iter"

	"github.com/google/uuid"
)

type idLookup interface {
	LookupID(id uuid.UUID) *Stub
}

type serviceLookup interface {
	LookupServiceAvailable(service, method string) (iter.Seq[*Stub], error)
}

type methodLookup interface {
	HasMethodAvailable(method string) bool
	LookupMethodAvailable(method string) iter.Seq[*Stub]
}

type stubLookup interface {
	idLookup
	serviceLookup
	methodLookup
}

type searcherIDLookup struct {
	searcher *searcher
}

type searcherSessionFallbackServiceLookup struct {
	searcher *searcher
	session  string
}

type searcherSessionFallbackMethodLookup struct {
	searcher *searcher
	session  string
}

type searcherLookupProvider interface {
	build(s *searcher, session string) *searcherLookup
}

type searcherLookupFactory struct {
	newID      func(*searcher) idLookup
	newService func(*searcher, string) serviceLookup
	newMethod  func(*searcher, string) methodLookup
}

type searcherLookup struct {
	id      idLookup
	service serviceLookup
	method  methodLookup
}

var (
	_ stubLookup    = (*searcherLookup)(nil)
	_ idLookup      = (*searcherIDLookup)(nil)
	_ serviceLookup = (*searcherSessionFallbackServiceLookup)(nil)
	_ methodLookup  = (*searcherSessionFallbackMethodLookup)(nil)
)

func (s *searcher) lookup(session string) *searcherLookup {
	s.lookupMu.RLock()
	lookup, ok := s.lookupCache[session]
	s.lookupMu.RUnlock()

	if ok {
		return lookup
	}

	s.lookupMu.Lock()
	defer s.lookupMu.Unlock()

	if lookup, ok = s.lookupCache[session]; ok {
		return lookup
	}

	lookup = s.lookupProvider.build(s, session)
	s.lookupCache[session] = lookup

	return lookup
}

func defaultSearcherLookupFactory() searcherLookupFactory {
	return searcherLookupFactory{
		newID: func(s *searcher) idLookup {
			return &searcherIDLookup{searcher: s}
		},
		newService: func(s *searcher, session string) serviceLookup {
			return &searcherSessionFallbackServiceLookup{
				searcher: s,
				session:  session,
			}
		},
		newMethod: func(s *searcher, session string) methodLookup {
			return &searcherSessionFallbackMethodLookup{
				searcher: s,
				session:  session,
			}
		},
	}
}

func (f searcherLookupFactory) build(s *searcher, session string) *searcherLookup {
	return &searcherLookup{
		id:      f.newID(s),
		service: f.newService(s, session),
		method:  f.newMethod(s, session),
	}
}

func (l *searcherIDLookup) LookupID(id uuid.UUID) *Stub {
	return l.searcher.findByID(id)
}

func (l *searcherSessionFallbackServiceLookup) LookupServiceAvailable(service, method string) (iter.Seq[*Stub], error) {
	seq, err := l.searcher.storage.findAllAvailable(service, method, l.session)
	if err != nil {
		return nil, err
	}

	return l.searcher.filterNotExhaustedSeq(seq, l.session), nil
}

func (l *searcherSessionFallbackMethodLookup) LookupMethodAvailable(method string) iter.Seq[*Stub] {
	if !l.searcher.storage.hasMethodAvailable(method, l.session) {
		return func(func(*Stub) bool) {}
	}

	return l.searcher.filterNotExhaustedSeq(l.searcher.storage.findByMethodAvailable(method, l.session), l.session)
}

func (l *searcherSessionFallbackMethodLookup) HasMethodAvailable(method string) bool {
	return l.searcher.storage.hasMethodAvailable(method, l.session)
}

func (l *searcherLookup) LookupID(id uuid.UUID) *Stub {
	return l.id.LookupID(id)
}

func (l *searcherLookup) LookupServiceAvailable(service, method string) (iter.Seq[*Stub], error) {
	return l.service.LookupServiceAvailable(service, method)
}

func (l *searcherLookup) LookupMethodAvailable(method string) iter.Seq[*Stub] {
	return l.method.LookupMethodAvailable(method)
}

func (l *searcherLookup) HasMethodAvailable(method string) bool {
	return l.method.HasMethodAvailable(method)
}
