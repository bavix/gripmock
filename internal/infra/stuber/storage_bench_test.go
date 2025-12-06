package stuber //nolint:testpackage

import (
	"iter"
	"testing"

	"github.com/google/uuid"
)

//nolint:gochecknoinits
func init() {
	uuid.EnableRandPool()
}

func BenchmarkStorageValues(b *testing.B) {
	items := make([]Value, 0, b.N)
	for range b.N {
		items = append(items, &testItem{id: uuid.New(), left: "A", right: "B"})
	}

	s := newStorage()
	s.upsert(items...)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		for range s.values() { //nolint:revive
		}
	}
}

func BenchmarkStorageFindAll(b *testing.B) {
	items := make([]Value, 0, b.N)
	for range b.N {
		items = append(items, &testItem{id: uuid.New(), left: "A", right: "B"})
	}

	s := newStorage()
	s.upsert(items...)

	var all iter.Seq[Value]

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		all, _ = s.findAll("A", "B")
		for range all { //nolint:revive
		}
	}
}

func BenchmarkStorageFindByID(b *testing.B) {
	items := make([]Value, 0, b.N)
	for range b.N {
		items = append(items, &testItem{id: uuid.New(), left: "A", right: "B"})
	}

	s := newStorage()
	s.upsert(items...)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = s.findByID(uuid.New())
	}
}

func BenchmarkStorageDel(b *testing.B) {
	items := make([]Value, 0, b.N)
	for range b.N {
		items = append(items, &testItem{id: uuid.New(), left: "A", right: "B"})
	}

	s := newStorage()
	s.upsert(items...)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = s.del(uuid.New())
	}
}

func BenchmarkStoragePosByN(b *testing.B) {
	s := newStorage()
	s.upsert(&testItem{id: uuid.New(), left: "A", right: "B"})

	b.ReportAllocs()

	for b.Loop() {
		_, _ = s.posByN("A", "B")
	}
}

func BenchmarkStoragePos(b *testing.B) {
	s := newStorage()

	left := s.id("A")
	right := s.id("B")

	b.ReportAllocs()

	for b.Loop() {
		_ = s.pos(left, right)
	}
}

func BenchmarkStorageLeftID(b *testing.B) {
	s := newStorage()
	s.upsert(&testItem{id: uuid.New(), left: "A", right: "B"})

	b.ReportAllocs()

	for b.Loop() {
		_ = s.id("A")
	}
}

func BenchmarkStorageLeftIDOrNew(b *testing.B) {
	s := newStorage()

	b.ReportAllocs()

	for b.Loop() {
		_ = s.id(uuid.NewString())
	}
}

func BenchmarkStorageRightID(b *testing.B) {
	s := newStorage()
	s.upsert(&testItem{id: uuid.New(), left: "A", right: "B"})

	b.ReportAllocs()

	for b.Loop() {
		_ = s.id("B")
	}
}

func BenchmarkStorageRightIDOrNew(b *testing.B) {
	s := newStorage()

	b.ReportAllocs()

	for b.Loop() {
		_ = s.id(uuid.NewString())
	}
}

func BenchmarkStorageFindAllSorted(b *testing.B) {
	items := make([]Value, 0, b.N)
	for i := range b.N {
		items = append(items, &testItem{
			id:    uuid.New(),
			left:  "A",
			right: "B",
			value: i % 100, // Different scores for sorting
		})
	}

	s := newStorage()
	s.upsert(items...)

	var all iter.Seq[Value]

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		all, _ = s.findAll("A", "B")
		for range all { //nolint:revive
		}
	}
}

func BenchmarkStorageUpsert(b *testing.B) {
	items := make([]Value, 0, b.N)
	for i := range b.N {
		items = append(items, &testItem{
			id:    uuid.New(),
			left:  "A",
			right: "B",
			value: i % 100,
		})
	}

	s := newStorage()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		s.upsert(items...)
	}
}
