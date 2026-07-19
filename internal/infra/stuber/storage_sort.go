package stuber

import (
	"bytes"
	"container/heap"
	"log"
	"slices"

	"github.com/google/uuid"
)

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

func sortedCopy(stubs []*Stub) []*Stub {
	sorted := make([]*Stub, len(stubs))
	copy(sorted, stubs)
	slices.SortFunc(sorted, compareStubsByPriorityAndID)

	return sorted
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

func collectAvailableSorted(indexBuckets map[uint64]map[string][]*Stub, indexes []uint64, session string) []*Stub {
	if len(indexes) == 0 {
		return nil
	}

	var total int

	for _, index := range indexes {
		buckets := indexBuckets[index]

		total += len(buckets[""])
		if session != "" {
			total += len(buckets[session])
		}
	}

	if total == 0 {
		return nil
	}

	result := make([]*Stub, 0, total)

	for _, index := range indexes {
		buckets := indexBuckets[index]

		result = append(result, buckets[""]...)
		if session != "" {
			result = append(result, buckets[session]...)
		}
	}

	slices.SortFunc(result, compareStubsByPriorityAndID)

	return result
}

func compareStubsByPriorityAndID(a, b *Stub) int {
	if a.Priority != b.Priority {
		return b.Priority - a.Priority
	}

	return bytes.Compare(a.ID[:], b.ID[:])
}
