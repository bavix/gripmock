package stuber

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBudgerigarListFilterSortPaginate(t *testing.T) {
	t.Parallel()

	b := NewBudgerigar()
	b.PutMany(
		&Stub{Service: "svc.A", Method: "Ping", Priority: 10, Source: "proxy", Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.A", Method: "Pong", Priority: 5, Source: "rest", Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.B", Method: "Ping", Priority: 1, Source: "file", Input: InputData{}, Output: Output{}},
	)

	stubs, total := b.List(ListOptions{
		Source:  "proxy",
		Service: "svc.A",
		Sort:    ListSortPriorityDesc,
		Limit:   1,
		Offset:  0,
	})

	require.Equal(t, 1, total)
	require.Len(t, stubs, 1)
	require.Equal(t, "proxy", stubs[0].Source)
	require.Equal(t, "svc.A", stubs[0].Service)
	require.Equal(t, "Ping", stubs[0].Method)
}

func TestBudgerigarListFilterByEmptySession(t *testing.T) {
	t.Parallel()

	b := NewBudgerigar()
	b.PutMany(
		&Stub{Service: "svc.A", Method: "Ping", Session: "", Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.A", Method: "Ping", Session: "s1", Input: InputData{}, Output: Output{}},
	)

	stubs, total := b.List(ListOptions{SessionSet: true, Session: ""})

	require.Equal(t, 1, total)
	require.Len(t, stubs, 1)
	require.Empty(t, stubs[0].Session)
}

func TestBudgerigarListFilters(t *testing.T) {
	t.Parallel()

	b := NewBudgerigar()
	b.PutMany(
		&Stub{Service: "svc.A", Method: "Ping", Source: "proxy", Session: "", Priority: 30, Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.A", Method: "Ping", Source: "rest", Session: "s1", Priority: 20, Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.A", Method: "Pong", Source: "rest", Session: "", Priority: 10, Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.B", Method: "Ping", Source: "file", Session: "s2", Priority: 5, Input: InputData{}, Output: Output{}},
	)

	t.Run("no filters", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{})
		require.Equal(t, 4, total)
		require.Len(t, stubs, 4)
	})

	t.Run("source filter", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{Source: "rest"})
		require.Equal(t, 2, total)
		require.Len(t, stubs, 2)

		for _, stub := range stubs {
			require.Equal(t, "rest", stub.Source)
		}
	})

	t.Run("service and method filter", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{Service: "svc.A", Method: "Ping"})
		require.Equal(t, 2, total)
		require.Len(t, stubs, 2)

		for _, stub := range stubs {
			require.Equal(t, "svc.A", stub.Service)
			require.Equal(t, "Ping", stub.Method)
		}
	})

	t.Run("session filter enabled", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{SessionSet: true, Session: "s1"})
		require.Equal(t, 1, total)
		require.Len(t, stubs, 1)
		require.Equal(t, "s1", stubs[0].Session)
	})

	t.Run("session ignored when not set", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{Session: "s1", SessionSet: false})
		require.Equal(t, 4, total)
		require.Len(t, stubs, 4)
	})
}

func TestBudgerigarListSorting(t *testing.T) {
	t.Parallel()

	b := newListSortPaginateFixture()

	t.Run("default sort is priority desc", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{})
		require.Equal(t, 3, total)
		require.Equal(t, 3, stubs[0].Priority)
		require.Equal(t, 2, stubs[1].Priority)
		require.Equal(t, 1, stubs[2].Priority)
	})

	t.Run("priority asc", func(t *testing.T) {
		t.Parallel()

		stubs, _ := b.List(ListOptions{Sort: ListSortPriorityAsc})
		require.Equal(t, 1, stubs[0].Priority)
		require.Equal(t, 2, stubs[1].Priority)
		require.Equal(t, 3, stubs[2].Priority)
	})

	t.Run("service asc then method asc", func(t *testing.T) {
		t.Parallel()

		stubs, _ := b.List(ListOptions{Sort: ListSortServiceAsc})
		require.Equal(t, "svc.A", stubs[0].Service)
		require.Equal(t, "A", stubs[0].Method)
		require.Equal(t, "svc.A", stubs[1].Service)
		require.Equal(t, "C", stubs[1].Method)
		require.Equal(t, "svc.B", stubs[2].Service)
	})

	t.Run("method asc then service asc", func(t *testing.T) {
		t.Parallel()

		stubs, _ := b.List(ListOptions{Sort: ListSortMethodAsc})
		require.Equal(t, "A", stubs[0].Method)
		require.Equal(t, "B", stubs[1].Method)
		require.Equal(t, "C", stubs[2].Method)
	})
}

func TestBudgerigarListPagination(t *testing.T) {
	t.Parallel()

	b := newListSortPaginateFixture()

	t.Run("negative offset and limit", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{Offset: -10, Limit: 2})
		require.Equal(t, 3, total)
		require.Len(t, stubs, 2)
		require.Equal(t, 3, stubs[0].Priority)
		require.Equal(t, 2, stubs[1].Priority)
	})

	t.Run("offset out of range", func(t *testing.T) {
		t.Parallel()

		stubs, total := b.List(ListOptions{Offset: 100})
		require.Equal(t, 3, total)
		require.Empty(t, stubs)
	})
}

func newListSortPaginateFixture() *Budgerigar {
	b := NewBudgerigar()
	b.PutMany(
		&Stub{Service: "svc.B", Method: "B", Priority: 3, Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.A", Method: "C", Priority: 1, Input: InputData{}, Output: Output{}},
		&Stub{Service: "svc.A", Method: "A", Priority: 2, Input: InputData{}, Output: Output{}},
	)

	return b
}
