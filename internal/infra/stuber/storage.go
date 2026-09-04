package stuber

import (
	"errors"
	"iter"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/zeebo/xxh3"
)

// ErrLeftNotFound is returned when the left value is not found.
var ErrLeftNotFound = errors.New("left not found")

// ErrRightNotFound is returned when the right value is not found.
var ErrRightNotFound = errors.New("right not found")

// storage is responsible for managing search results with enhanced
// performance and memory efficiency. It supports concurrent access
// through the use of a read-write mutex.
//
// Fields:
// - mu: Ensures safe concurrent access to the storage.
// - lefts: A map that tracks unique left values by their hashed IDs.
// - itemsByID: Provides quick access to items by their unique UUIDs.
type storage struct {
	mu           sync.RWMutex
	lefts        map[uint32]struct{}
	methodSorted map[uint32]map[string][]*Stub
	itemSorted   map[uint64]map[string][]*Stub
	equalsIndex  map[uint64]map[string]map[string][]*Stub
	unindexed    map[uint64][]*Stub
	itemsByID    map[uuid.UUID]*Stub
	sessions     map[string]int
}

// newStorage creates a new instance of the storage struct.
func newStorage() *storage {
	return &storage{
		lefts:        make(map[uint32]struct{}),
		methodSorted: make(map[uint32]map[string][]*Stub),
		itemSorted:   make(map[uint64]map[string][]*Stub),
		equalsIndex:  make(map[uint64]map[string]map[string][]*Stub),
		unindexed:    make(map[uint64][]*Stub),
		itemsByID:    make(map[uuid.UUID]*Stub),
		sessions:     make(map[string]int),
	}
}

// clear resets the storage.
func (s *storage) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lefts = make(map[uint32]struct{})
	s.methodSorted = make(map[uint32]map[string][]*Stub)
	s.itemSorted = make(map[uint64]map[string][]*Stub)
	s.equalsIndex = make(map[uint64]map[string]map[string][]*Stub)
	s.unindexed = make(map[uint64][]*Stub)
	s.itemsByID = make(map[uuid.UUID]*Stub)
	s.sessions = make(map[string]int)
}

// findByMethodAvailable retrieves method stubs visible for session.
func (s *storage) findByMethodAvailable(method, session string) iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		s.mu.RLock()
		methodID := s.id(method)
		global := s.methodSorted[methodID][""]

		var all []*Stub

		if session == "" {
			all = slices.Clone(global)
		} else {
			sessionStubs := s.methodSorted[methodID][session]
			all = make([]*Stub, 0, len(global)+len(sessionStubs))
			all = append(all, global...)
			all = append(all, sessionStubs...)
		}

		s.mu.RUnlock()

		slices.SortFunc(all, compareStubsByPriorityAndID)

		for _, stub := range all {
			if !yield(stub) {
				return
			}
		}
	}
}

func (s *storage) hasMethodAvailable(method, session string) bool {
	methodID := s.id(method)

	s.mu.RLock()
	defer s.mu.RUnlock()

	buckets := s.methodSorted[methodID]
	if len(buckets[""]) > 0 {
		return true
	}

	if session == "" {
		return false
	}

	return len(buckets[session]) > 0
}

// findAllAvailable retrieves stubs by service/method visible for session.
func (s *storage) findAllAvailable(left, right, session string) (iter.Seq[*Stub], error) {
	indexes, err := s.posByPN(left, right)
	if err != nil {
		return nil, err
	}

	return func(yield func(*Stub) bool) {
		s.mu.RLock()
		snapshot := detachStubs(collectAvailableSorted(s.itemSorted, indexes, session))
		s.mu.RUnlock()

		for _, stub := range snapshot {
			if !yield(stub) {
				return
			}
		}
	}, nil
}

// values returns an iterator sequence of all Stub items stored in the storage.
func (s *storage) size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.itemsByID)
}

func (s *storage) values() iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		s.mu.RLock()
		snapshot := make([]*Stub, 0, len(s.itemsByID))

		for _, v := range s.itemsByID {
			snapshot = append(snapshot, v)
		}

		s.mu.RUnlock()

		for _, v := range snapshot {
			if !yield(v) {
				return
			}
		}
	}
}

// findAll retrieves all Stub items that match the given left and right names,
// sorted by score in descending order.
func (s *storage) findAll(left, right string) (iter.Seq[*Stub], error) {
	indexes, err := s.posByPN(left, right)
	if err != nil {
		return nil, err
	}

	return func(yield func(*Stub) bool) {
		s.yieldSortedValues(indexes, yield)
	}, nil
}

// posByPN resolves ID pairs for a service/method name pair, trying the full
// service name first and then the package-truncated form. Returns
// ErrLeftNotFound (service) or ErrRightNotFound (method) — part of the contract.
func (s *storage) posByPN(left, right string) ([]uint64, error) {
	var resolvedIDs []uint64

	var lastErr error

	id, err := s.posByN(left, right)
	if err == nil {
		resolvedIDs = append(resolvedIDs, id)
	} else {
		lastErr = err
	}

	// Retry with the package prefix stripped (e.g. pkg.Svc → Svc).
	if dotIndex := strings.LastIndex(left, "."); dotIndex != -1 {
		truncatedLeft := left[dotIndex+1:]

		id, err := s.posByN(truncatedLeft, right)
		if err == nil {
			resolvedIDs = append(resolvedIDs, id)
		} else if errors.Is(err, ErrRightNotFound) && len(resolvedIDs) == 0 {
			return nil, err
		}
	}

	if len(resolvedIDs) == 0 {
		return nil, lastErr
	}

	return resolvedIDs, nil
}

// findByID retrieves the Stub associated with the given UUID from the storage.
func (s *storage) findByID(key uuid.UUID) *Stub {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.itemsByID[key]
}

// findByIDs retrieves the Stubs associated with the given UUIDs from the storage.
func (s *storage) findByIDs(ids iter.Seq[uuid.UUID]) iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		for id := range ids {
			if v, ok := s.itemsByID[id]; ok {
				if !yield(v) {
					return
				}
			}
		}
	}
}

func (s *storage) id(value string) uint32 {
	return uint32(xxh3.HashString(value)) //nolint:gosec
}

func (s *storage) pos(a, b uint32) uint64 {
	return uint64(a)<<32 | uint64(b)
}

func (s *storage) posByN(leftName, rightName string) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	leftID := s.id(leftName)
	if _, exists := s.lefts[leftID]; !exists {
		return 0, ErrLeftNotFound
	}

	rightID := s.id(rightName)
	key := s.pos(leftID, rightID)

	if _, exists := s.itemSorted[key]; !exists {
		return 0, ErrRightNotFound
	}

	return key, nil
}
