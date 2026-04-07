package stuber

import (
	"iter"

	"github.com/google/uuid"
)

// stubStorage defines the interface for stub storage operations.
// Both *storage and *storageWithInternal implement this interface.
//
//nolint:interfacebloat
type stubStorage interface {
	upsert(stubs ...*Stub) []uuid.UUID
	del(ids ...uuid.UUID) int
	findByID(id uuid.UUID) *Stub
	findByIDs(ids iter.Seq[uuid.UUID]) iter.Seq[*Stub]
	findAll(service, method string) (iter.Seq[*Stub], error)
	findByMethodAvailable(method, session string) iter.Seq[*Stub]
	hasMethodAvailable(method, session string) bool
	findAllAvailable(service, method, session string) (iter.Seq[*Stub], error)
	posByPN(left, right string) ([]uint64, error)
	values() iter.Seq[*Stub]
	sessionsList() []string
	clear()
}

// InternalStubStorage defines the interface for adding internal stubs.
// Internal stubs are hidden from user-facing APIs and take precedence in matching.
// Only ADD operation is allowed - no delete, no clear (controlled by Budgerigar internally).
type InternalStubStorage interface {
	// PutInternal adds stubs to the internal storage (hidden from users).
	// This is the ONLY way to add internal stubs.
	PutInternal(stubs ...*Stub) []uuid.UUID

	// FindByIDInternal finds a stub by ID in internal storage.
	FindByIDInternal(id uuid.UUID) *Stub

	// FindAllAvailable finds stubs by service/method in internal storage.
	FindAllAvailable(service, method, session string) (iter.Seq[*Stub], error)

	// FindByMethodAvailable finds stubs by method in internal storage.
	FindByMethodAvailable(method, session string) iter.Seq[*Stub]

	// HasMethodAvailable checks if method exists in internal storage.
	HasMethodAvailable(method, session string) bool
}

// internalStorageAdapter wraps *storage to implement InternalStubStorage.
// Only exposes add-only operations - no delete/clear to protect internal stubs.
type internalStorageAdapter struct {
	storage *storage
}

func newInternalStorageAdapter(s *storage) *internalStorageAdapter {
	return &internalStorageAdapter{storage: s}
}

func (a *internalStorageAdapter) PutInternal(stubs ...*Stub) []uuid.UUID {
	return a.storage.upsert(stubs...)
}

func (a *internalStorageAdapter) FindByIDInternal(id uuid.UUID) *Stub {
	return a.storage.findByID(id)
}

func (a *internalStorageAdapter) FindAllAvailable(service, method, session string) (iter.Seq[*Stub], error) {
	return a.storage.findAllAvailable(service, method, session)
}

func (a *internalStorageAdapter) FindByMethodAvailable(method, session string) iter.Seq[*Stub] {
	return a.storage.findByMethodAvailable(method, session)
}

func (a *internalStorageAdapter) HasMethodAvailable(method, session string) bool {
	return a.storage.hasMethodAvailable(method, session)
}
