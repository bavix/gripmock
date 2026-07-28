package stuber

import (
	"iter"

	"github.com/google/uuid"
)

// storageWithInternal wraps external storage and dedicated internal storage.
// Internal stubs are stored separately with special handling:
// - Internal stubs are queried first in matching
// - Internal stubs are hidden from user-facing APIs (All, Used, Unused, etc.)
type storageWithInternal struct {
	*storage

	internalBase *storage
	internal     InternalStubStorage
}

// newStorageWithInternal creates a storage that supports internal stubs.
// It automatically initializes internal gripmock health stubs with NOT_SERVING status.
func newStorageWithInternal() *storageWithInternal {
	external := newStorage()
	internal := newStorage()

	storageWithInternal := &storageWithInternal{
		storage:      external,
		internalBase: internal,
		internal:     newInternalStorageAdapter(internal),
	}

	// Initialize internal gripmock health stubs with NOT_SERVING
	SetupGripmockHealthStubs(storageWithInternal.internal)

	return storageWithInternal
}

// Internal returns the internal storage interface for adding/managing internal stubs.
//
//nolint:ireturn
func (s *storageWithInternal) Internal() InternalStubStorage {
	return s.internal
}

// findAllAvailable, findByMethodAvailable and hasMethodAvailable are intentionally
// NOT overridden here: they promote to the embedded external *storage, so the
// general search never sees internal stubs. Internal stubs (the gripmock health
// status) are a separate index consulted ONLY by searcher.matchInternalStubs,
// which the gRPC health service reaches via Query.WithInternalStubs(). This keeps
// internal state invisible to list/get/search/inspect/verify/MCP.

// findByID returns stub by ID from external storage only.
// Internal stubs are hidden from direct ID lookup to prevent collisions with user stubs.
func (s *storageWithInternal) findByID(id uuid.UUID) *Stub {
	return s.storage.findByID(id)
}

// values returns only external stubs (internal hidden from API).
func (s *storageWithInternal) values() iter.Seq[*Stub] {
	return s.storage.values()
}

// sessionsList returns only external sessions.
func (s *storageWithInternal) sessionsList() []string {
	return s.storage.sessionsList()
}

// clear clears both storage and internal storage.
func (s *storageWithInternal) clear() {
	s.storage.clear()
	s.internalBase.clear()
}

// delBySession deletes all external stubs belonging to the given session.
// Internal stubs are not session-scoped and are never deleted by session.
func (s *storageWithInternal) delBySession(session string) int {
	return s.storage.delBySession(session)
}

// findByIDs returns stubs by IDs from storage.
func (s *storageWithInternal) findByIDs(ids iter.Seq[uuid.UUID]) iter.Seq[*Stub] {
	return s.storage.findByIDs(ids)
}

// posByPN returns position by service/method from storage.
func (s *storageWithInternal) posByPN(left, right string) ([]uint64, error) {
	return s.storage.posByPN(left, right)
}
