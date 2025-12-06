package stuber

import (
	"container/heap"
	"errors"
	"iter"
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

// Value is a type used to store the result of a search.
type Value interface {
	Key() uuid.UUID
	Left() string
	Right() string
	Score() int // Score determines the order of values when sorting
}

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
	mu        sync.RWMutex
	lefts     map[uint32]struct{}
	items     map[uint64]map[uuid.UUID]Value
	itemsByID map[uuid.UUID]Value
}

// newStorage creates a new instance of the storage struct.
func newStorage() *storage {
	return &storage{
		lefts:     make(map[uint32]struct{}),
		items:     make(map[uint64]map[uuid.UUID]Value),
		itemsByID: make(map[uuid.UUID]Value),
	}
}

// clear resets the storage.
func (s *storage) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lefts = make(map[uint32]struct{})
	s.items = make(map[uint64]map[uuid.UUID]Value)
	s.itemsByID = make(map[uuid.UUID]Value)
}

// values returns an iterator sequence of all Value items stored in the
// storage.
func (s *storage) values() iter.Seq[Value] {
	return func(yield func(Value) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		for _, v := range s.itemsByID {
			if !yield(v) {
				return
			}
		}
	}
}

// findAll retrieves all Value items that match the given left and right names,
// sorted by score in descending order.
func (s *storage) findAll(left, right string) (iter.Seq[Value], error) {
	indexes, err := s.posByPN(left, right)
	if err != nil {
		return nil, err
	}

	return func(yield func(Value) bool) {
		s.yieldSortedValues(indexes, yield)
	}, nil
}

// yieldSortedValues yields values sorted by score in descending order,
// minimizing memory allocations and maximizing iterator usage.
func (s *storage) yieldSortedValues(indexes []uint64, yield func(Value) bool) {
	s.yieldSortedValuesOptimized(indexes, yield)
}

// yieldSortedValuesOptimized is an ultra-optimized version with minimal allocations.
//
//nolint:gocognit,cyclop,funlen,nestif
func (s *storage) yieldSortedValuesOptimized(indexes []uint64, yield func(Value) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ultra-fast path: single index with single value (most common case)
	if len(indexes) == 1 {
		if m, exists := s.items[indexes[0]]; exists && len(m) == 1 {
			for _, v := range m {
				if !yield(v) {
					return
				}
			}

			return
		}
	}

	// Ultra-fast path: empty result
	if len(indexes) == 0 {
		return
	}

	// Pre-count total items for optimal allocation
	totalItems := s.countItemsFast(indexes)

	// Ultra-fast path: single item
	if totalItems == 1 {
		for _, index := range indexes {
			if m, exists := s.items[index]; exists {
				for _, v := range m {
					if !yield(v) {
						return
					}
				}

				return
			}
		}

		return
	}

	// Fast path: small dataset (≤smallItemsThreshold items) - use ultra-simple iteration
	if totalItems <= smallItemsThreshold {
		items := make([]Value, 0, totalItems)

		// Collect items
		for _, index := range indexes {
			if m, exists := s.items[index]; exists {
				for _, v := range m {
					items = append(items, v)
				}
			}
		}

		// Ultra-simple sort for 2-3 items
		if len(items) == twoItemsThreshold {
			if items[0].Score() < items[1].Score() {
				items[0], items[1] = items[1], items[0]
			}
		} else if len(items) == smallItemsThreshold {
			// Manual sort for smallItemsThreshold items (faster than bubble sort)
			if items[0].Score() < items[1].Score() {
				items[0], items[1] = items[1], items[0]
			}

			if items[1].Score() < items[2].Score() {
				items[1], items[2] = items[2], items[1]
			}

			if items[0].Score() < items[1].Score() {
				items[0], items[1] = items[1], items[0]
			}
		}

		for _, v := range items {
			if !yield(v) {
				return
			}
		}

		return
	}

	// Large dataset - use heap-based sorting for O(N log N) performance
	s.yieldSortedValuesHeap(indexes, yield)
}

// sortItem represents a value with its score for sorting.
type sortItem struct {
	value Value
	score int
}

// countItemsFast provides ultra-fast counting of items without collecting them.
func (s *storage) countItemsFast(indexes []uint64) int {
	total := 0

	for _, index := range indexes {
		if m, exists := s.items[index]; exists {
			total += len(m)
		}
	}

	return total
}

// scoreHeap implements heap.Interface for sorting by score.
type scoreHeap []sortItem

func (h *scoreHeap) Len() int           { return len(*h) }
func (h *scoreHeap) Less(i, j int) bool { return (*h)[i].score > (*h)[j].score }
func (h *scoreHeap) Swap(i, j int)      { (*h)[i], (*h)[j] = (*h)[j], (*h)[i] }
func (h *scoreHeap) Push(x any) {
	if item, ok := x.(sortItem); ok {
		*h = append(*h, item)
	}
}

func (h *scoreHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]

	return x
}

// yieldSortedValuesHeap uses heap-based sorting for O(N log N) performance.
//
//nolint:gocognit,cyclop,funlen
func (s *storage) yieldSortedValuesHeap(indexes []uint64, yield func(Value) bool) {
	// Ultra-fast path: single index with single value (most common case)
	if len(indexes) == 1 {
		if m, exists := s.items[indexes[0]]; exists && len(m) == 1 {
			for _, v := range m {
				if !yield(v) {
					return
				}
			}

			return
		}
	}

	// Fast path: single index with multiple values
	//nolint:nestif
	if len(indexes) == 1 {
		if m, exists := s.items[indexes[0]]; exists {
			// Use slice-based sorting for small collections (faster than heap)
			if len(m) <= smallCollectionThreshold {
				items := make([]sortItem, 0, len(m))
				for _, v := range m {
					items = append(items, sortItem{value: v, score: v.Score()})
				}
				// Sort in descending order
				for i := range len(items) - 1 {
					for j := i + 1; j < len(items); j++ {
						if items[i].score < items[j].score {
							items[i], items[j] = items[j], items[i]
						}
					}
				}

				for _, item := range items {
					if !yield(item.value) {
						return
					}
				}

				return
			}
		}
	}

	// Use heap for complex cases
	h := &scoreHeap{}
	heap.Init(h)

	// Pre-allocate heap capacity for better performance
	totalItems := s.countItemsFast(indexes)
	if totalItems > 0 {
		*h = make(scoreHeap, 0, totalItems)
	}

	// Collect elements in heap
	for _, index := range indexes {
		if m, exists := s.items[index]; exists {
			for _, v := range m {
				heap.Push(h, sortItem{value: v, score: v.Score()})
			}
		}
	}

	// Extract elements in descending score order
	for h.Len() > 0 {
		item, ok := heap.Pop(h).(sortItem)
		if !ok {
			continue
		}

		if !yield(item.value) {
			return
		}
	}
}

// posByPN attempts to resolve IDs for a given left and right name pair.
// It first tries to resolve the full left name with the right name, and then
// attempts to resolve using a truncated version of the left name if necessary.
//
// Parameters:
// - left: The left name for matching.
// - right: The right name for matching.
//
// Returns:
// - [][2]uint64: A slice of resolved ID pairs.
// - error: An error if no IDs were resolved.
func (s *storage) posByPN(left, right string) ([]uint64, error) {
	// Initialize a slice to store the resolved IDs.
	var resolvedIDs []uint64

	// Attempt to resolve the full left name with the right name.
	id, err := s.posByN(left, right)
	if err == nil {
		// Append the resolved ID to the slice.
		resolvedIDs = append(resolvedIDs, id)
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
			// and no IDs were resolved.
			return nil, err
		}
	}

	// Return an error if no IDs were resolved.
	if len(resolvedIDs) == 0 {
		// Return the original error if we have it.
		return nil, err
	}

	// Return the resolved IDs.
	return resolvedIDs, nil
}

// findByID retrieves the Stub value associated with the given UUID from the
// storage.
//
// Parameters:
// - key: The UUID of the Stub value to retrieve.
//
// Returns:
// - Value: The Stub value associated with the given UUID, or nil if not found.
func (s *storage) findByID(key uuid.UUID) Value { //nolint:ireturn
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.itemsByID[key]
}

// findByIDs retrieves the Stub values associated with the given UUIDs from the
// storage.
//
// Returns:
//   - iter.Seq[Value]: The Stub values associated with the given UUIDs, or nil if
//     not found.
func (s *storage) findByIDs(ids iter.Seq[uuid.UUID]) iter.Seq[Value] {
	return func(yield func(Value) bool) {
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

// upsert inserts or updates the given Value items in storage.
// Optimized for minimal allocations and maximum performance.
func (s *storage) upsert(values ...Value) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}

	// Pre-allocate with exact size to minimize allocations
	results := make([]uuid.UUID, len(values))

	s.mu.Lock()
	defer s.mu.Unlock()

	// Process all values in a single pass
	for i, v := range values {
		results[i] = v.Key()

		// Calculate IDs directly without string interning
		leftID := s.id(v.Left())
		rightID := s.id(v.Right())
		index := s.pos(leftID, rightID)

		// Initialize the map at the index if it doesn't exist.
		if s.items[index] == nil {
			s.items[index] = make(map[uuid.UUID]Value, 1)
		}

		// Insert or update the value in the storage.
		s.items[index][v.Key()] = v
		s.itemsByID[v.Key()] = v
		s.lefts[leftID] = struct{}{}
	}

	return results
}

// del deletes the Stub values with the given UUIDs from the storage.
// It returns the number of Stub values that were successfully deleted.
func (s *storage) del(keys ...uuid.UUID) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := 0

	for _, key := range keys {
		if v, ok := s.itemsByID[key]; ok {
			pos := s.pos(s.id(v.Left()), s.id(v.Right()))

			if m, exists := s.items[pos]; exists {
				delete(m, key)
				delete(s.itemsByID, key)

				deleted++

				if len(m) == 0 {
					delete(s.items, pos)
				}
			}
		}
	}

	return deleted
}

// Global LRU cache for string hashes with size limit.
//
//nolint:gochecknoglobals
var globalStringCache *lru.Cache[string, uint32]

//nolint:gochecknoinits
func init() {
	var err error
	// Create LRU cache with size limit of stringCacheSize entries
	globalStringCache, err = lru.New[string, uint32](stringCacheSize)
	if err != nil {
		panic("failed to create string hash cache: " + err.Error())
	}
}

func (s *storage) id(value string) uint32 {
	// Try to get from cache first
	if hash, exists := globalStringCache.Get(value); exists {
		return hash
	}

	// Calculate hash and store in cache
	hash := uint32(xxh3.HashString(value)) //nolint:gosec
	globalStringCache.Add(value, hash)

	return hash
}

// clearStringHashCache clears the string hash cache.
func clearStringHashCache() {
	globalStringCache.Purge()
}

// getStringHashCacheStats returns cache statistics.
func getStringHashCacheStats() (int, int) {
	return globalStringCache.Len(), stringCacheSize // Fixed capacity
}

// ClearAllCaches clears all LRU caches (for testing purposes).
func ClearAllCaches() {
	clearStringHashCache()
	clearRegexCache()
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

	key := s.pos(leftID, s.id(rightName))

	if _, exists := s.items[key]; !exists {
		return 0, ErrRightNotFound
	}

	return key, nil
}
