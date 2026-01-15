package stuber //nolint:testpackage

import (
	"iter"
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	id          uuid.UUID
	left, right string
	value       int
}

func (t testItem) Key() uuid.UUID {
	return t.id
}

func (t testItem) Left() string {
	return t.left
}

func (t testItem) Right() string {
	return t.right
}

func (t testItem) Score() int {
	return t.value
}

func TestAdd(t *testing.T) {
	t.Parallel()

	s := newStorage()
	s.upsert(
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1"},
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1"},
		&testItem{id: uuid.New(), left: "Greeter2", right: "SayHello2"},
		&testItem{id: uuid.New(), left: "Greeter3", right: "SayHello2"},
		&testItem{id: uuid.New(), left: "Greeter4", right: "SayHello3"},
		&testItem{id: uuid.New(), left: "Greeter5", right: "SayHello3"},
	)

	require.Len(t, s.items, 5)
	require.Len(t, s.itemsByID, 6)
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	s := newStorage()
	s.upsert(&testItem{id: id, left: "Greeter", right: "SayHello"})

	require.Len(t, s.items, 1)
	require.Len(t, s.itemsByID, 1)

	v := s.findByID(id)
	require.NotNil(t, v)

	val, ok := v.(*testItem)
	require.True(t, ok)
	require.Equal(t, 0, val.value)

	s.upsert(&testItem{id: id, left: "Greeter", right: "SayHello", value: 42})

	require.Len(t, s.items, 1)
	require.Len(t, s.itemsByID, 1)

	v = s.findByID(id)
	require.NotNil(t, v)

	val, ok = v.(*testItem)
	require.True(t, ok)
	require.Equal(t, 42, val.value)
}

func TestFindByID(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("00000000-0000-0001-0000-000000000000")

	s := newStorage()
	require.Nil(t, s.findByID(id))

	s.upsert(
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1"},
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1"},
		&testItem{id: uuid.New(), left: "Greeter2", right: "SayHello2"},
		&testItem{id: uuid.New(), left: "Greeter3", right: "SayHello2"},
		&testItem{id: uuid.New(), left: "Greeter4", right: "SayHello3"},
		&testItem{id: uuid.New(), left: "Greeter5", right: "SayHello3"},
		&testItem{id: id, left: "Greeter1", right: "SayHello3"},
	)

	require.Len(t, s.items, 6)
	require.Len(t, s.itemsByID, 7)

	val := s.findByID(id)
	require.NotNil(t, val)
	require.Equal(t, id, val.Key())
}

func TestFindAll(t *testing.T) {
	t.Parallel()

	s := newStorage()
	s.upsert(
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1"},
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1"},
		&testItem{id: uuid.New(), left: "Greeter2", right: "SayHello2"},
		&testItem{id: uuid.New(), left: "Greeter3", right: "SayHello2"},
		&testItem{id: uuid.New(), left: "Greeter4", right: "SayHello3"},
		&testItem{id: uuid.New(), left: "Greeter5", right: "SayHello3"},
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello3"},
	)

	collect := func(seq iter.Seq[Value]) []Value {
		var res []Value
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
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	s.upsert(
		&testItem{id: id1, left: "A", right: "B"},
		&testItem{id: id2, left: "C", right: "D"},
		&testItem{id: id3, left: "E", right: "F"},
	)

	t.Run("existing IDs", func(t *testing.T) {
		t.Parallel()

		results := make([]Value, 0, 2)
		for v := range s.findByIDs(maps.Keys(map[uuid.UUID]struct{}{id1: {}, id2: {}})) {
			results = append(results, v)
		}

		require.Len(t, results, 2)
	})

	t.Run("mixed IDs", func(t *testing.T) {
		t.Parallel()

		results := make([]Value, 0, 1)
		for v := range s.findByIDs(maps.Keys(map[uuid.UUID]struct{}{id1: {}, uuid.Nil: {}})) {
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
		&testItem{id: id1, left: "Greeter1", right: "SayHello1"},
		&testItem{id: id2, left: "Greeter2", right: "SayHello2"},
		&testItem{id: id3, left: "Greeter3", right: "SayHello3"},
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
	item1 := &testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1", value: 10}
	item2 := &testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1", value: 30}
	item3 := &testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1", value: 20}
	item4 := &testItem{id: uuid.New(), left: "Greeter2", right: "SayHello2", value: 50}

	s.upsert(item1, item2, item3, item4)

	collect := func(seq iter.Seq[Value]) []Value {
		var res []Value
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
		require.Equal(t, 30, results[0].Score())
		require.Equal(t, 20, results[1].Score())
		require.Equal(t, 10, results[2].Score())
	})

	t.Run("single item", func(t *testing.T) {
		t.Parallel()

		seq, err := s.findAll("Greeter2", "SayHello2")
		require.NoError(t, err)

		results := collect(seq)
		require.Len(t, results, 1)
		require.Equal(t, 50, results[0].Score())
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
		&testItem{id: uuid.New(), left: "Greeter1", right: "SayHello1"},
		&testItem{id: uuid.New(), left: "Greeter2", right: "SayHello2"},
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
	item1 := &testItem{id: uuid.New(), left: "A", right: "B"}
	item2 := &testItem{id: uuid.New(), left: "C", right: "D"}
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
	item1 := &testItem{id: uuid.New(), left: "A", right: "B"}
	item2 := &testItem{id: uuid.New(), left: "C", right: "D"}
	item3 := &testItem{id: uuid.New(), left: "E", right: "F"}
	s.upsert(item1, item2, item3)

	// Test finding by IDs
	ids := []uuid.UUID{item1.id, item2.id}

	found := make([]Value, 0, len(ids))
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
	notFound := make([]Value, 0, 2)
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
	s.upsert(&testItem{id: uuid.New(), left: "A", right: "B"})

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
