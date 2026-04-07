package stuber

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

// PriorityMultiplier is used to boost stub priority in ranking calculations.
// Higher values give more weight to explicit priority settings.
const PriorityMultiplier = 10.0

// parallelProcessingThreshold is the threshold for using parallel processing.
const parallelProcessingThreshold = 100

// ErrServiceNotFound is returned when the service is not found.
var ErrServiceNotFound = errors.New("service not found")

// ErrMethodNotFound is returned when the method is not found.
var ErrMethodNotFound = errors.New("method not found")

// ErrStubNotFound is returned when the stub is not found.
var ErrStubNotFound = errors.New("stub not found")

// callCountKey identifies a stub's match count per session. Session empty = global count.
type callCountKey struct {
	id      uuid.UUID
	session string
}

// searcher is a struct that manages the storage of search results.
//
// It contains a mutex for concurrent access, a map to store and retrieve
// used stubs by their UUID (and session for isolation), and a pointer to the storage struct.
type searcher struct {
	mu              sync.RWMutex
	lookupMu        sync.RWMutex
	stubCallCount   map[callCountKey]int // count of matches per stub+session (for Times limit)
	storage         stubStorage
	internalStorage InternalStubStorage
	lookupProvider  searcherLookupProvider
	lookupCache     map[string]*searcherLookup
	processStrategy processStubsStrategy
	matcher         matchStrategy
	ranker          rankStrategy
}

// Result holds the search result: exact match (Found) or best similar (Similar).
type Result struct {
	found   *Stub // The exact match found in the search
	similar *Stub // The most similar match found
}

// Found returns the exact match found in the search.
func (r *Result) Found() *Stub {
	return r.found
}

// Similar returns the most similar match found in the search.
func (r *Result) Similar() *Stub {
	return r.similar
}
