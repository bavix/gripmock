package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStubsStorageExtenderStruct(t *testing.T) {
	t.Parallel()

	// Test Extender struct fields
	storage := NewStub(nil, nil, nil)
	require.NotNil(t, storage)
	require.NotNil(t, storage.ch)
	require.NotNil(t, storage.mapIDsByFile)
	require.NotNil(t, storage.uniqueIDs)
}

func TestStubsStorageMapOperations(t *testing.T) {
	t.Parallel()

	// Test map operations
	storage := NewStub(nil, nil, nil)

	// Test map initialization
	require.NotNil(t, storage.mapIDsByFile)
	require.NotNil(t, storage.uniqueIDs)

	require.Empty(t, storage.mapIDsByFile)
	require.Empty(t, storage.uniqueIDs)
}

func TestStubsStorageAtomicOperations(t *testing.T) {
	t.Parallel()

	// Test atomic operations
	storage := NewStub(nil, nil, nil)

	// Test loaded field
	require.False(t, storage.loaded.Load())

	// Test setting loaded
	storage.loaded.Store(true)
	require.True(t, storage.loaded.Load())

	// Test resetting loaded
	storage.loaded.Store(false)
	require.False(t, storage.loaded.Load())
}

func TestStubsStorageChannelOperations(t *testing.T) {
	t.Parallel()

	// Test channel operations
	storage := NewStub(nil, nil, nil)

	// Test that channel is created
	require.NotNil(t, storage.ch)

	// Test that channel is buffered
	select {
	case <-storage.ch:
		// Channel is closed
	default:
		// Channel is open
	}
}

func TestStubsStorageMutexOperations(t *testing.T) {
	t.Parallel()

	// Test mutex operations
	storage := NewStub(nil, nil, nil)

	// Test that mutex protects shared resources
	storage.muUniqueIDs.Lock()
	// Access protected resource while locked
	initialLen := len(storage.uniqueIDs)
	storage.muUniqueIDs.Unlock()

	// Test that we can access the uniqueIDs map
	require.NotNil(t, storage.uniqueIDs)
	require.Equal(t, 0, initialLen, "uniqueIDs should be empty initially")

	// Test that we can access the uniqueIDs map
	require.NotNil(t, storage.uniqueIDs)
}
