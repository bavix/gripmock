package stuber

import (
	"bytes"
	"container/heap"
	"errors"
	"iter"
	"log"
	"maps"
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
}

// newStorage creates a new instance of the storage struct.
func newStorage() *storage {
	return &storage{
		lefts:        make(map[uint32]struct{}),
		methodSorted: make(map[uint32]map[string][]*Stub),
		items:        make(map[uint64]map[uuid.UUID]*Stub),
		itemSorted:   make(map[uint64]map[string][]*Stub),
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
		if session == "" {
			for _, stub := range s.methodSorted[methodID][""] {
				if !yield(stub) {
					return
				}
			}

			return
		}

		yieldMergedSorted(s.methodSorted[methodID][""], s.methodSorted[methodID][session], yield)
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

// yieldSortedValues yields values sorted by score in descending order,
// minimizing memory allocations and maximizing iterator usage.
func (s *storage) yieldSortedValues(indexes []uint64, yield func(*Stub) bool) {
	s.yieldSortedValuesOptimized(indexes, yield)
}

// yieldSortedValuesOptimized is an ultra-optimized version with minimal allocations.
func (s *storage) yieldSortedValuesOptimized(indexes []uint64, yield func(*Stub) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tryYieldSingleItem(indexes, yield) {
		return
	}

	totalItems := s.countItemsFast(indexes)
	if totalItems <= smallItemsThreshold {
		s.yieldSmallItemsSorted(indexes, totalItems, yield)

		return
	}

	s.yieldSortedValuesHeap(indexes, yield)
}

func (s *storage) tryYieldSingleItem(indexes []uint64, yield func(*Stub) bool) bool {
	if len(indexes) != 1 {
		return false
	}

	m, exists := s.items[indexes[0]]
	if !exists || len(m) != 1 {
		return false
	}

	for _, v := range m {
		if !yield(v) {
			return true
		}
	}

	return true
}

func (s *storage) yieldSmallItemsSorted(indexes []uint64, totalItems int, yield func(*Stub) bool) {
	items := make([]*Stub, 0, totalItems)
	for _, index := range indexes {
		if m, exists := s.items[index]; exists {
			for _, v := range m {
				items = append(items, v)
			}
		}
	}

	sortSmallItemsByPriority(items)

	for _, v := range items {
		if !yield(v) {
			return
		}
	}
}

func sortSmallItemsByPriority(items []*Stub) {
	switch len(items) {
	case twoItemsThreshold:
		if items[0].Priority < items[1].Priority {
			items[0], items[1] = items[1], items[0]
		}
	case smallItemsThreshold:
		if items[0].Priority < items[1].Priority {
			items[0], items[1] = items[1], items[0]
		}

		if items[1].Priority < items[2].Priority {
			items[1], items[2] = items[2], items[1]
		}

		if items[0].Priority < items[1].Priority {
			items[0], items[1] = items[1], items[0]
		}
	}
}

// sortItem represents a stub with its score for sorting.
type sortItem struct {
	stub  *Stub
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
	item, ok := x.(sortItem)
	if !ok {
		log.Printf("[gripmock] scoreHeap.Push: expected sortItem, got %T", x)

		return
	}

	*h = append(*h, item)
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
//nolint:cyclop,gocognit
func (s *storage) yieldSortedValuesHeap(indexes []uint64, yield func(*Stub) bool) {
	// Fast path: single index with multiple values
	//nolint:nestif
	if len(indexes) == 1 {
		if m, exists := s.items[indexes[0]]; exists {
			// Use slice-based sorting for small collections (faster than heap)
			if len(m) <= smallCollectionThreshold {
				items := make([]sortItem, 0, len(m))
				for _, v := range m {
					items = append(items, sortItem{stub: v, score: v.Priority})
				}

				slices.SortFunc(items, func(a, b sortItem) int { return b.score - a.score }) // descending

				for _, item := range items {
					if !yield(item.stub) {
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
				heap.Push(h, sortItem{stub: v, score: v.Priority})
			}
		}
	}

	// Extract elements in descending score order
	for h.Len() > 0 {
		x := heap.Pop(h)

		item, ok := x.(sortItem)
		if !ok {
			log.Printf("[gripmock] scoreHeap.Pop: expected sortItem, got %T", x)

			continue
		}

		if !yield(item.stub) {
			return
		}
	}
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

// upsert inserts or updates the given Stubs in storage.
// Optimized for minimal allocations and maximum performance.
func (s *storage) upsert(values ...*Stub) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}

	// Pre-allocate with exact size to minimize allocations
	results := make([]uuid.UUID, len(values))

	s.mu.Lock()
	defer s.mu.Unlock()

	// Process all values in a single pass (direct field access for performance)
	for i, v := range values {
		results[i] = v.ID

		if old, exists := s.itemsByID[v.ID]; exists {
			s.removeStubIndexes(old)
		}

		leftID := s.id(v.Service)
		rightID := s.id(v.Method)
		index := s.pos(leftID, rightID)

		if s.items[index] == nil {
			s.items[index] = make(map[uuid.UUID]*Stub, 1)
		}

		s.items[index][v.ID] = v
		s.upsertSessionIndex(s.itemSorted, index, v.Session, v)
		s.upsertMethodSessionIndex(rightID, v.Session, v)
		s.incrementSession(v.Session)
		s.itemsByID[v.ID] = v
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
			s.removeStubIndexes(v)
			delete(s.itemsByID, key)

			deleted++
		}
	}

	return deleted
}

func (s *storage) removeStubIndexes(stub *Stub) {
	pos := s.pos(s.id(stub.Service), s.id(stub.Method))

	if m, exists := s.items[pos]; exists {
		delete(m, stub.ID)

		if len(m) == 0 {
			delete(s.items, pos)
		}
	}

	s.removeSessionIndex(s.itemSorted, pos, stub.Session, stub.ID)
	methodID := s.id(stub.Method)
	s.removeMethodSessionIndex(methodID, stub.Session, stub.ID)
	s.decrementSession(stub.Session)
}

func (s *storage) incrementSession(session string) {
	if session == "" {
		return
	}

	s.sessions[session]++
}

func (s *storage) decrementSession(session string) {
	if session == "" {
		return
	}

	next := s.sessions[session] - 1
	if next <= 0 {
		delete(s.sessions, session)

		return
	}

	s.sessions[session] = next
}

func (s *storage) sessionsList() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := slices.Collect(maps.Keys(s.sessions))
	if sessions == nil {
		return []string{}
	}

	slices.Sort(sessions)

	return sessions
}

func (s *storage) upsertSessionIndex(
	sorted map[uint64]map[string][]*Stub,
	key uint64,
	session string,
	stub *Stub,
) {
	sortedBuckets := sorted[key]
	if sortedBuckets == nil {
		sortedBuckets = make(map[string][]*Stub)
		sorted[key] = sortedBuckets
	}

	sortedBuckets[session] = insertSortedStub(sortedBuckets[session], stub)
}

func (s *storage) upsertMethodSessionIndex(key uint32, session string, stub *Stub) {
	sortedBuckets := s.methodSorted[key]
	if sortedBuckets == nil {
		sortedBuckets = make(map[string][]*Stub)
		s.methodSorted[key] = sortedBuckets
	}

	sortedBuckets[session] = insertSortedStub(sortedBuckets[session], stub)
}

func (s *storage) removeSessionIndex(
	sorted map[uint64]map[string][]*Stub,
	key uint64,
	session string,
	id uuid.UUID,
) {
	sortedBuckets, exists := sorted[key]
	if !exists {
		return
	}

	sortedBuckets[session] = removeSortedStubByID(sortedBuckets[session], id)
	if len(sortedBuckets[session]) == 0 {
		delete(sortedBuckets, session)
	}

	if len(sortedBuckets) == 0 {
		delete(sorted, key)
	}
}

func (s *storage) removeMethodSessionIndex(key uint32, session string, id uuid.UUID) {
	sortedBuckets, exists := s.methodSorted[key]
	if !exists {
		return
	}

	sortedBuckets[session] = removeSortedStubByID(sortedBuckets[session], id)
	if len(sortedBuckets[session]) == 0 {
		delete(sortedBuckets, session)
	}

	if len(sortedBuckets) == 0 {
		delete(s.methodSorted, key)
	}
}

func insertSortedStub(stubs []*Stub, stub *Stub) []*Stub {
	idx, _ := slices.BinarySearchFunc(stubs, stub, compareStubsByPriorityAndID)
	stubs = append(stubs, nil)
	copy(stubs[idx+1:], stubs[idx:])
	stubs[idx] = stub

	return stubs
}

func removeSortedStubByID(stubs []*Stub, id uuid.UUID) []*Stub {
	for i, stub := range stubs {
		if stub.ID == id {
			copy(stubs[i:], stubs[i+1:])

			return stubs[:len(stubs)-1]
		}
	}

	return stubs
}

func yieldMergedSorted(global, session []*Stub, yield func(*Stub) bool) {
	i := 0
	j := 0

	for i < len(global) && j < len(session) {
		if compareStubsByPriorityAndID(global[i], session[j]) <= 0 {
			if !yield(global[i]) {
				return
			}

			i++

			continue
		}

		if !yield(session[j]) {
			return
		}

		j++
	}

	for i < len(global) {
		if !yield(global[i]) {
			return
		}

		i++
	}

	for j < len(session) {
		if !yield(session[j]) {
			return
		}

		j++
	}
}

func collectAvailableSorted(indexBuckets map[uint64]map[string][]*Stub, indexes []uint64, session string) []*Stub {
	if len(indexes) == 0 {
		return nil
	}

	var merged []*Stub

	for _, index := range indexes {
		buckets := indexBuckets[index]
		if len(buckets) == 0 {
			continue
		}

		indexStubs := buckets[""]
		if session != "" {
			indexStubs = mergeSortedSlices(indexStubs, buckets[session])
		}

		if len(indexStubs) == 0 {
			continue
		}

		if len(merged) == 0 {
			merged = indexStubs

			continue
		}

		merged = mergeSortedSlices(merged, indexStubs)
	}

	return merged
}

func mergeSortedSlices(left, right []*Stub) []*Stub {
	if len(left) == 0 {
		return right
	}

	if len(right) == 0 {
		return left
	}

	merged := make([]*Stub, 0, len(left)+len(right))
	i := 0
	j := 0

	for i < len(left) && j < len(right) {
		if compareStubsByPriorityAndID(left[i], right[j]) <= 0 {
			merged = append(merged, left[i])
			i++

			continue
		}

		merged = append(merged, right[j])
		j++
	}

	merged = append(merged, left[i:]...)
	merged = append(merged, right[j:]...)

	return merged
}

func compareStubsByPriorityAndID(a, b *Stub) int {
	if a.Priority != b.Priority {
		return b.Priority - a.Priority
	}

	return bytes.Compare(a.ID[:], b.ID[:])
}

// Global LRU cache for string hashes with size limit.
//
//nolint:gochecknoglobals
var globalStringCache *lru.Cache[string, uint32]

// initStringCache initializes the global string hash cache. Used by init and tests.
// Does not panic on error; logs and sets globalStringCache to nil.
func initStringCache(size int) {
	cache, err := lru.New[string, uint32](size)
	if err != nil {
		log.Printf("[gripmock] failed to create string hash cache: %v", err)

		globalStringCache = nil

		return
	}

	globalStringCache = cache
}

//nolint:gochecknoinits
func init() {
	initStringCache(stringCacheSize)
}

func (s *storage) id(value string) uint32 {
	if globalStringCache != nil {
		if hash, exists := globalStringCache.Get(value); exists {
			return hash
		}
	}

	hash := uint32(xxh3.HashString(value)) //nolint:gosec
	if globalStringCache != nil {
		globalStringCache.Add(value, hash)
	}

	return hash
}

// clearStringHashCache clears the string hash cache (for testing).
func clearStringHashCache() {
	if globalStringCache != nil {
		globalStringCache.Purge()
	}
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

	rightID := s.id(rightName)
	key := s.pos(leftID, rightID)

	if _, exists := s.items[key]; !exists {
		return 0, ErrRightNotFound
	}

	return key, nil
}
