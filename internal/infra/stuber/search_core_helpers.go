package stuber

import (
	"errors"
	"iter"

	"github.com/google/uuid"
)

func collectStubs(seq iter.Seq[*Stub]) []*Stub {
	var result []*Stub

	for stub := range seq {
		result = append(result, stub)
	}

	return result
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
	if _, err := s.storage.posByPN(service, method); err != nil {
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
