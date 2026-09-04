package stuber

import (
	"bytes"
	"slices"

	"github.com/google/uuid"
)

// yieldSortedValues yields the stubs for the given indexes in priority order.
//
// Every per-(index, session) bucket is kept sorted at write time, so this
// merges already-sorted sequences instead of collecting and re-sorting the
// whole set on each call.
func (s *storage) yieldSortedValues(indexes []uint64, yield func(*Stub) bool) {
	s.mu.RLock()
	snapshot := detachStubs(mergeSortedParts(s.sortedParts(indexes)))
	s.mu.RUnlock()

	for _, stub := range snapshot {
		if !yield(stub) {
			return
		}
	}
}

func detachStubs(stubs []*Stub) []*Stub {
	if len(stubs) == 0 {
		return nil
	}

	return slices.Clone(stubs)
}

// sortedParts collects the sorted stub slices held for the given indexes,
// across every session.
func (s *storage) sortedParts(indexes []uint64) [][]*Stub {
	var parts [][]*Stub

	for _, index := range indexes {
		for _, stubs := range s.itemSorted[index] {
			if len(stubs) > 0 {
				parts = append(parts, stubs)
			}
		}
	}

	return parts
}

// mergeSortedParts merges pre-sorted slices into one sorted slice. A single
// part is returned as-is, so the common case copies nothing.
func mergeSortedParts(parts [][]*Stub) []*Stub {
	switch len(parts) {
	case 0:
		return nil
	case 1:
		return parts[0]
	}

	merged := parts[0]
	for _, part := range parts[1:] {
		merged = mergeSortedStubs(merged, part)
	}

	return merged
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

// collectAvailableSorted returns available stubs across indexes/session in
// priority+ID order. Each source bucket is already sorted (storage.upsert
// sorts touched buckets once, at write time), so the common case -- a single
// index and no session -- returns the stored bucket directly with no copy or
// sort. Rarer multi-bucket cases (package-prefix fallback matched both index
// forms, or a session combined with the global bucket) merge the already-
// sorted sources in O(total) instead of re-sorting from scratch.
func collectAvailableSorted(indexBuckets map[uint64]map[string][]*Stub, indexes []uint64, session string) []*Stub {
	if len(indexes) == 0 {
		return nil
	}

	var parts [][]*Stub

	for _, index := range indexes {
		buckets := indexBuckets[index]

		if global := buckets[""]; len(global) > 0 {
			parts = append(parts, global)
		}

		if session != "" {
			if sessionStubs := buckets[session]; len(sessionStubs) > 0 {
				parts = append(parts, sessionStubs)
			}
		}
	}

	switch len(parts) {
	case 0:
		return nil
	case 1:
		return parts[0]
	default:
		merged := parts[0]
		for _, part := range parts[1:] {
			merged = mergeSortedStubs(merged, part)
		}

		return merged
	}
}

// mergeSortedStubs merges two slices already sorted by
// compareStubsByPriorityAndID into one sorted slice.
func mergeSortedStubs(a, b []*Stub) []*Stub {
	result := make([]*Stub, 0, len(a)+len(b))

	var i, j int

	for i < len(a) && j < len(b) {
		if compareStubsByPriorityAndID(a[i], b[j]) <= 0 {
			result = append(result, a[i])
			i++
		} else {
			result = append(result, b[j])
			j++
		}
	}

	result = append(result, a[i:]...)
	result = append(result, b[j:]...)

	return result
}

func compareStubsByPriorityAndID(a, b *Stub) int {
	if a.Priority != b.Priority {
		return b.Priority - a.Priority
	}

	return bytes.Compare(a.ID[:], b.ID[:])
}
