package stuber //nolint:testpackage

import (
	"iter"
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newTestStub(service, method string, priority int) *Stub {
	return &Stub{
		ID:       uuid.New(),
		Service:  service,
		Method:   method,
		Priority: priority,
	}
}

func newTestStubWithID(id uuid.UUID, service, method string, priority int) *Stub {
	return &Stub{
		ID:       id,
		Service:  service,
		Method:   method,
		Priority: priority,
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()

	s := newStorage()
	s.upsert(
		newTestStub("Greeter1", "SayHello1", 0),
		newTestStub("Greeter1", "SayHello1", 0),
		newTestStub("Greeter2", "SayHello2", 0),
		newTestStub("Greeter3", "SayHello2", 0),
		newTestStub("Greeter4", "SayHello3", 0),
		newTestStub("Greeter5", "SayHello3", 0),
	)

	require.Len(t, s.items, 5)
	require.Len(t, s.itemsByID, 6)
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	s := newStorage()
	s.upsert(newTestStubWithID(id, "Greeter", "SayHello", 0))

	require.Len(t, s.items, 1)
	require.Len(t, s.itemsByID, 1)

	v := s.findByID(id)
	require.NotNil(t, v)
	require.Equal(t, 0, v.Priority)

	s.upsert(newTestStubWithID(id, "Greeter", "SayHello", 42))

	require.Len(t, s.items, 1)
	require.Len(t, s.itemsByID, 1)

	v = s.findByID(id)
	require.NotNil(t, v)
	require.Equal(t, 42, v.Priority)
}

func TestFindByID(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("00000000-0000-0001-0000-000000000000")

	s := newStorage()
	require.Nil(t, s.findByID(id))

	s.upsert(
		newTestStub("Greeter1", "SayHello1", 0),
		newTestStub("Greeter1", "SayHello1", 0),
		newTestStub("Greeter2", "SayHello2", 0),
		newTestStub("Greeter3", "SayHello2", 0),
		newTestStub("Greeter4", "SayHello3", 0),
		newTestStub("Greeter5", "SayHello3", 0),
		newTestStubWithID(id, "Greeter1", "SayHello3", 0),
	)

	require.Len(t, s.items, 6)
	require.Len(t, s.itemsByID, 7)

	val := s.findByID(id)
	require.NotNil(t, val)
	require.Equal(t, id, val.ID)
}

func TestFindAll(t *testing.T) {
	t.Parallel()

	s := newStorage()
	s.upsert(
		newTestStub("Greeter1", "SayHello1", 0),
		newTestStub("Greeter1", "SayHello1", 0),
		newTestStub("Greeter2", "SayHello2", 0),
		newTestStub("Greeter3", "SayHello2", 0),
		newTestStub("Greeter4", "SayHello3", 0),
		newTestStub("Greeter5", "SayHello3", 0),
		newTestStub("Greeter1", "SayHello3", 0),
	)

	collect := func(seq iter.Seq[*Stub]) []*Stub {
		var res []*Stub
		for v := range seq {
			res = append(res, v)
		}

		return res
	}

	t.Run("Greeter1/SayHello1", func(t *testing.T) {
		t.Parallel()

		seq, err := s.findAll("Greeter1", "SayHello1")
		require.NoError(t, err)
		require.Len(t, collect(seq), 2)
	})

	t.Run("Greeter2/SayHello2", func(t *testing.T) {
		t.Parallel()

		seq, err := s.findAll("Greeter2", "SayHello2")
		require.NoError(t, err)
		require.Len(t, collect(seq), 1)
	})

	t.Run("Greeter3/SayHello2", func(t *testing.T) {
		t.Parallel()

		seq, err := s.findAll("Greeter3", "SayHello2")
		require.NoError(t, err)
		require.Len(t, collect(seq), 1)
	})

	t.Run("Greeter3/SayHello3", func(t *testing.T) {
		t.Parallel()

		_, err := s.findAll("Greeter3", "SayHello3")
		require.ErrorIs(t, err, ErrRightNotFound)
	})
}

func TestFindByIDs(t *testing.T) {
	t.Parallel()

	s := newStorage()
	stub1 := newTestStubWithID(uuid.New(), "A", "B", 0)
	stub2 := newTestStubWithID(uuid.New(), "C", "D", 0)
	stub3 := newTestStubWithID(uuid.New(), "E", "F", 0)
	s.upsert(stub1, stub2, stub3)

	t.Run("existing IDs", func(t *testing.T) {
		t.Parallel()

		results := make([]*Stub, 0, 2)
		for v := range s.findByIDs(maps.Keys(map[uuid.UUID]struct{}{stub1.ID: {}, stub2.ID: {}})) {
			results = append(results, v)
		}

		require.Len(t, results, 2)
	})

	t.Run("mixed IDs", func(t *testing.T) {
		t.Parallel()

		results := make([]*Stub, 0, 1)
		for v := range s.findByIDs(maps.Keys(map[uuid.UUID]struct{}{stub1.ID: {}, uuid.Nil: {}})) {
			results = append(results, v)
		}

		require.Len(t, results, 1)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()

	s := newStorage()

	s.upsert(
		newTestStubWithID(id1, "Greeter1", "SayHello1", 0),
		newTestStubWithID(id2, "Greeter2", "SayHello2", 0),
		newTestStubWithID(id3, "Greeter3", "SayHello3", 0),
	)

	require.Equal(t, 0, s.del())
	require.Len(t, s.items, 3)
	require.Len(t, s.itemsByID, 3)

	require.Equal(t, 1, s.del(id1))
	require.Len(t, s.items, 2)
	require.Len(t, s.itemsByID, 2)

	require.Equal(t, 2, s.del(id2, id3))
	require.Empty(t, s.items)
	require.Empty(t, s.itemsByID)

	require.Equal(t, 0, s.del(id1, id2, id3))
	require.Empty(t, s.items)
	require.Empty(t, s.itemsByID)
}

func TestFindAllSorted(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// Create items with different scores
	item1 := newTestStub("Greeter1", "SayHello1", 10)
	item2 := newTestStub("Greeter1", "SayHello1", 30)
	item3 := newTestStub("Greeter1", "SayHello1", 20)
	item4 := newTestStub("Greeter2", "SayHello2", 50)

	s.upsert(item1, item2, item3, item4)

	collect := func(seq iter.Seq[*Stub]) []*Stub {
		var res []*Stub
		for v := range seq {
			res = append(res, v)
		}

		return res
	}

	t.Run("sorted by score descending", func(t *testing.T) {
		t.Parallel()

		seq, err := s.findAll("Greeter1", "SayHello1")
		require.NoError(t, err)

		results := collect(seq)
		require.Len(t, results, 3)

		// Should be sorted by score in descending order: 30, 20, 10
		require.Equal(t, 30, results[0].Priority)
		require.Equal(t, 20, results[1].Priority)
		require.Equal(t, 10, results[2].Priority)
	})

	t.Run("single item", func(t *testing.T) {
		t.Parallel()

		seq, err := s.findAll("Greeter2", "SayHello2")
		require.NoError(t, err)

		results := collect(seq)
		require.Len(t, results, 1)
		require.Equal(t, 50, results[0].Priority)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := s.findAll("Greeter3", "SayHello3")
		require.ErrorIs(t, err, ErrLeftNotFound)
	})

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()

		_, err := s.findAll("Greeter1", "SayHello2")
		require.ErrorIs(t, err, ErrRightNotFound)
	})
}

func TestClear(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// Add some items
	s.upsert(
		newTestStub("Greeter1", "SayHello1", 0),
		newTestStub("Greeter2", "SayHello2", 0),
	)

	require.Len(t, s.items, 2)
	require.Len(t, s.itemsByID, 2)
	require.Len(t, s.lefts, 2)

	// Clear storage
	s.clear()

	require.Empty(t, s.items)
	require.Empty(t, s.itemsByID)
	require.Empty(t, s.lefts)
}

func TestStorageValues(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// Initially empty
	var count int
	for range s.values() {
		count++
	}

	require.Equal(t, 0, count)

	// Add items
	item1 := newTestStub("A", "B", 0)
	item2 := newTestStub("C", "D", 0)
	s.upsert(item1, item2)

	// Count items
	count = 0
	for range s.values() {
		count++
	}

	require.Equal(t, 2, count)

	// Test early return in iterator
	count = 0
	for range s.values() {
		count++
		if count == 1 {
			break
		}
	}

	require.Equal(t, 1, count)
}

func TestStorageFindByIDs(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// Add items
	item1 := newTestStub("A", "B", 0)
	item2 := newTestStub("C", "D", 0)
	item3 := newTestStub("E", "F", 0)
	s.upsert(item1, item2, item3)

	// Test finding by IDs
	ids := []uuid.UUID{item1.ID, item2.ID}

	found := make([]*Stub, 0, len(ids))
	for v := range s.findByIDs(func(yield func(uuid.UUID) bool) {
		for _, id := range ids {
			if !yield(id) {
				return
			}
		}
	}) {
		found = append(found, v)
	}

	require.Len(t, found, 2)

	// Test finding by non-existent IDs
	notFound := make([]*Stub, 0, 2)
	for v := range s.findByIDs(func(yield func(uuid.UUID) bool) {
		yield(uuid.New())
		yield(uuid.New())
	}) {
		notFound = append(notFound, v)
	}

	require.Empty(t, notFound)
}

func TestStorageFindAll_EmptyResult(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// Add items
	s.upsert(newTestStub("A", "B", 0))

	_, err := s.findAll("NonExistent", "Method")
	require.ErrorIs(t, err, ErrLeftNotFound)
}

func TestStorageUpsert_Empty(t *testing.T) {
	t.Parallel()

	s := newStorage()

	result := s.upsert()
	require.Nil(t, result)
}

func TestStorageDel_NonExistent(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// Test deleting non-existent items
	deleted := s.del(uuid.New(), uuid.New())
	require.Equal(t, 0, deleted)
}

func TestStorageFindAll_HeapPath_SingleIndexManyItems(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// Create 12 items with same left/right to hit heap path (single index, >smallCollectionThreshold)
	items := make([]*Stub, 12)
	for i := range 12 {
		items[i] = newTestStub("HeapService", "HeapMethod", i+1)
	}

	s.upsert(items...)

	seq, err := s.findAll("HeapService", "HeapMethod")
	require.NoError(t, err)

	results := make([]*Stub, 0, 12)
	for v := range seq {
		results = append(results, v)
	}

	require.Len(t, results, 12)
	// Should be sorted descending by score
	for i := range len(results) - 1 {
		require.GreaterOrEqual(t, results[i].Priority, results[i+1].Priority)
	}
}

func TestStorageFindAll_HeapPath_MultipleIndexes(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// posByPN with "pkg.Svc" returns both "pkg.Svc" and "Svc" indexes when both exist
	s.upsert(
		newTestStub("pkg.Greeter", "SayHello", 1),
		newTestStub("pkg.Greeter", "SayHello", 2),
		newTestStub("Greeter", "SayHello", 10),
		newTestStub("Greeter", "SayHello", 20),
	)

	seq, err := s.findAll("pkg.Greeter", "SayHello")
	require.NoError(t, err)

	results := make([]*Stub, 0, 4)
	for v := range seq {
		results = append(results, v)
	}

	require.Len(t, results, 4)
	require.Equal(t, 20, results[0].Priority)
	require.Equal(t, 10, results[1].Priority)
	require.Equal(t, 2, results[2].Priority)
	require.Equal(t, 1, results[3].Priority)
}

func TestStorageFindAll_YieldEarlyExit(t *testing.T) {
	t.Parallel()

	s := newStorage()
	s.upsert(
		newTestStub("A", "B", 1),
		newTestStub("A", "B", 2),
		newTestStub("A", "B", 3),
	)

	seq, err := s.findAll("A", "B")
	require.NoError(t, err)

	count := 0
	for range seq {
		count++
		if count >= 2 {
			break
		}
	}

	require.Equal(t, 2, count)
}

func TestStorageFindByIDs_EarlyExit(t *testing.T) {
	t.Parallel()

	s := newStorage()
	item1 := newTestStub("A", "B", 0)
	item2 := newTestStub("C", "D", 0)
	s.upsert(item1, item2)

	ids := func(yield func(uuid.UUID) bool) {
		for _, id := range []uuid.UUID{item1.ID, item2.ID} {
			if !yield(id) {
				return
			}
		}
	}

	count := 0
	for range s.findByIDs(ids) {
		count++
		if count >= 1 {
			break
		}
	}

	require.Equal(t, 1, count)
}

func TestStorageFindAll_SliceSortPath(t *testing.T) {
	t.Parallel()

	s := newStorage()

	// 5 items in single index - hits yieldSortedValuesHeap slice path (4-10 items)
	for i := 5; i >= 1; i-- {
		s.upsert(newTestStub("SliceSvc", "SliceMethod", i))
	}

	seq, err := s.findAll("SliceSvc", "SliceMethod")
	require.NoError(t, err)

	results := make([]*Stub, 0, 5)
	for v := range seq {
		results = append(results, v)
	}

	require.Len(t, results, 5)

	for i := range len(results) - 1 {
		require.GreaterOrEqual(t, results[i].Priority, results[i+1].Priority)
	}
}

func TestStorageFindAll_SingleItemFastPath(t *testing.T) {
	t.Parallel()

	s := newStorage()
	s.upsert(newTestStub("Single", "Item", 42))

	seq, err := s.findAll("Single", "Item")
	require.NoError(t, err)

	// Drain fully to cover the return after yielding single item
	var count int
	for range seq {
		count++
	}

	require.Equal(t, 1, count)
}

func TestStorageFindAll_HeapPathEarlyExit(t *testing.T) {
	t.Parallel()

	s := newStorage()
	// 12 items to hit heap path
	for i := range 12 {
		s.upsert(newTestStub("HeapSvc", "HeapMethod", i))
	}

	seq, err := s.findAll("HeapSvc", "HeapMethod")
	require.NoError(t, err)

	count := 0
	for range seq {
		count++
		if count >= 2 {
			break
		}
	}

	require.Equal(t, 2, count)
}

func TestStorageFindAll_SlicePathEarlyExit(t *testing.T) {
	t.Parallel()

	s := newStorage()
	// 5 items - slice sort path in yieldSortedValuesHeap
	for i := range 5 {
		s.upsert(newTestStub("SliceSvc", "SliceMethod", i*10))
	}

	seq, err := s.findAll("SliceSvc", "SliceMethod")
	require.NoError(t, err)

	count := 0
	for range seq {
		count++
		if count >= 1 {
			break
		}
	}

	require.Equal(t, 1, count)
}

func TestMustCreateStringCache_PanicsOnInvalidSize(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		mustCreateStringCache(0)
	})
}

func TestStorageFindAll_ThreeItemSort(t *testing.T) {
	t.Parallel()

	s := newStorage()
	s.upsert(
		newTestStub("Tri", "Sort", 10),
		newTestStub("Tri", "Sort", 5),
		newTestStub("Tri", "Sort", 20),
	)
	seq, err := s.findAll("Tri", "Sort")
	require.NoError(t, err)

	results := make([]*Stub, 0, 3)
	for v := range seq {
		results = append(results, v)
	}

	require.Len(t, results, 3)
	// Must be sorted descending by priority
	require.Equal(t, 20, results[0].Priority)
	require.Equal(t, 10, results[1].Priority)
	require.Equal(t, 5, results[2].Priority)
}
