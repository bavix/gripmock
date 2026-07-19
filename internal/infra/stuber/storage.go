package stuber

import (
	"errors"
	"iter"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/zeebo/xxh3"
)

const (
	// smallCollectionThreshold is the threshold for using simple sorting instead of heap.
	smallCollectionThreshold = 10
	// smallItemsThreshold is the threshold for using simple sorting instead of heap.
	smallItemsThreshold = 3
	// twoItemsThreshold is the threshold for two items case.
	twoItemsThreshold = 2
	// stringCacheSize is the maximum number of string hashes to cache.
	stringCacheSize = 10000
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
// - items: Stores items by a composite key of hashed left and right IDs.
// - itemsByID: Provides quick access to items by their unique UUIDs.
type storage struct {
	mu           sync.RWMutex
	lefts        map[uint32]struct{}
	methodSorted map[uint32]map[string][]*Stub
	items        map[uint64]map[uuid.UUID]*Stub
	itemSorted   map[uint64]map[string][]*Stub
	itemsByID    map[uuid.UUID]*Stub
	sessions     map[string]int
	stringCache  *lru.Cache[string, uint32]
}

// newStorage creates a new instance of the storage struct.
func newStorage() *storage {
	cache, _ := lru.New[string, uint32](stringCacheSize)

	return &storage{
		lefts:        make(map[uint32]struct{}),
		methodSorted: make(map[uint32]map[string][]*Stub),
		items:        make(map[uint64]map[uuid.UUID]*Stub),
		itemSorted:   make(map[uint64]map[string][]*Stub),
		itemsByID:    make(map[uuid.UUID]*Stub),
		sessions:     make(map[string]int),
		stringCache:  cache,
	}
}

// clear resets the storage.
func (s *storage) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lefts = make(map[uint32]struct{})
	s.methodSorted = make(map[uint32]map[string][]*Stub)
	s.items = make(map[uint64]map[uuid.UUID]*Stub)
	s.itemSorted = make(map[uint64]map[string][]*Stub)
	s.itemsByID = make(map[uuid.UUID]*Stub)
	s.sessions = make(map[string]int)
}

// findByMethodAvailable retrieves method stubs visible for session.
func (s *storage) findByMethodAvailable(method, session string) iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		methodID := s.id(method)

		global := s.methodSorted[methodID][""]
		if session == "" {
			sorted := sortedCopy(global)
			for _, stub := range sorted {
				if !yield(stub) {
					return
				}
			}

			return
		}

		sessionStubs := s.methodSorted[methodID][session]
		all := make([]*Stub, 0, len(global)+len(sessionStubs))
		all = append(all, global...)
		all = append(all, sessionStubs...)
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
		defer s.mu.RUnlock()

		for _, stub := range collectAvailableSorted(s.itemSorted, indexes, session) {
			if !yield(stub) {
				return
			}
		}
	}, nil
}

// values returns an iterator sequence of all Stub items stored in the storage.
func (s *storage) values() iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		for _, v := range s.itemsByID {
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

// posByPN attempts to resolve IDs for a given left and right name pair.
// It first tries to resolve the full left name with the right name, and then
// attempts to resolve using a truncated version of the left name if necessary.
// Returns error if service or method is not found - this is part of the public contract.
//
// Parameters:
// - left: The left name for matching (service name).
// - right: The right name for matching (method name).
//
// Returns:
// - []uint64: A slice of resolved ID pairs.
// - error: ErrLeftNotFound (service not found) or ErrRightNotFound (method not found).
func (s *storage) posByPN(left, right string) ([]uint64, error) {
	// Initialize a slice to store the resolved IDs.
	var resolvedIDs []uint64

	// Track the last error for reporting
	var lastErr error

	// Attempt to resolve the full left name with the right name.
	id, err := s.posByN(left, right)
	if err == nil {
		// Append the resolved ID to the slice.
		resolvedIDs = append(resolvedIDs, id)
	} else {
		lastErr = err
	}

	// Check for a potential truncation point in the left name.
	if dotIndex := strings.LastIndex(left, "."); dotIndex != -1 {
		truncatedLeft := left[dotIndex+1:]

		// Attempt to resolve the truncated left name with the right name.
		id, err := s.posByN(truncatedLeft, right)
		if err == nil {
			// Append the resolved ID to the slice.
			resolvedIDs = append(resolvedIDs, id)
		} else if errors.Is(err, ErrRightNotFound) && len(resolvedIDs) == 0 {
			// Return an error if the right name was not found
			// and no IDs were resolved (even with truncated name).
			return nil, err
		}
	}

	// Return an error if no IDs were resolved.
	if len(resolvedIDs) == 0 {
		// Return the original error if we have it.
		return nil, lastErr
	}

	// Return the resolved IDs.
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
	if s.stringCache != nil {
		if hash, exists := s.stringCache.Get(value); exists {
			return hash
		}
	}

	hash := uint32(xxh3.HashString(value)) //nolint:gosec
	if s.stringCache != nil {
		s.stringCache.Add(value, hash)
	}

	return hash
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

	if _, exists := s.items[key]; !exists {
		return 0, ErrRightNotFound
	}

	return key, nil
}
