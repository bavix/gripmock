package stuber

import (
	"errors"
	"iter"
	"sync"

	"github.com/google/uuid"
)

// candidateBufPool recycles the candidate slices built for each search. The
// slice is sized by the number of stubs registered for one service/method, so
// with a large stub set every request would otherwise allocate (and grow,
// repeatedly) a multi-megabyte slice that is dead the moment the search
// returns.
//
//nolint:gochecknoglobals // process-wide buffer recycling has to be package state
var candidateBufPool = sync.Pool{
	New: func() any { return new([]*Stub) },
}

// collectStubs materializes seq into a freshly allocated slice. Use it when
// the result escapes the caller (API listings, inspect, etc.).
func collectStubs(seq iter.Seq[*Stub]) []*Stub {
	var result []*Stub

	for stub := range seq {
		result = append(result, stub)
	}

	return result
}

// collectStubsPooled materializes seq into a pooled slice. Callers MUST
// release the result with releaseStubs once done, and must not retain it (or
// any sub-slice) afterwards. Individual *Stub pointers stay valid.
func collectStubsPooled(seq iter.Seq[*Stub]) []*Stub {
	bufPtr, _ := candidateBufPool.Get().(*[]*Stub)
	result := (*bufPtr)[:0]

	for stub := range seq {
		result = append(result, stub)
	}

	*bufPtr = result

	return result
}

// releaseStubs returns a slice obtained from collectStubs to the pool.
func releaseStubs(stubs []*Stub) {
	if stubs == nil {
		return
	}

	clear(stubs)

	buf := stubs[:0]
	candidateBufPool.Put(&buf)
}

// wrap wraps an error with specific error types.
func (s *searcher) wrap(err error) error {
	if errors.Is(err, ErrLeftNotFound) {
		return ErrServiceNotFound
	}

	if errors.Is(err, ErrRightNotFound) {
		return ErrMethodNotFound
	}

	return err
}

func (s *searcher) ensureServiceMethodExists(service, method string) error {
	_, err := s.storage.posByPN(service, method)
	if err != nil {
		return s.wrap(err)
	}

	return nil
}

func (s *searcher) lookupVisibleByID(session string, id uuid.UUID) (*searcherLookup, *Stub) {
	lookup := s.lookup(session)
	found := lookup.LookupID(id)

	if found == nil || !s.isVisibleAndNotExhausted(found, session) {
		return lookup, nil
	}

	return lookup, found
}
