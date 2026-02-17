package history_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/history"
)

func TestMemoryStoreDeleteSession_RemovesOnlySessionRecords(t *testing.T) {
	t.Parallel()

	// Arrange
	store := &history.MemoryStore{}
	store.Record(history.CallRecord{Service: "svc", Method: "A", Session: "s1"})
	store.Record(history.CallRecord{Service: "svc", Method: "B", Session: "s2"})
	store.Record(history.CallRecord{Service: "svc", Method: "C", Session: ""})

	// Act
	deleted := store.DeleteSession("s1")

	// Assert
	require.Equal(t, 1, deleted)

	all := store.All()
	require.Len(t, all, 2)
	require.Equal(t, "s2", all[0].Session)
	require.Empty(t, all[1].Session)
}

func TestMemoryStoreDeleteSession_EmptySessionNoop(t *testing.T) {
	t.Parallel()

	// Arrange
	store := &history.MemoryStore{}
	store.Record(history.CallRecord{Service: "svc", Method: "A", Session: "s1"})

	// Act
	deleted := store.DeleteSession("")

	// Assert
	require.Equal(t, 0, deleted)
	require.Len(t, store.All(), 1)
}
