package stuber

import (
	"github.com/google/uuid"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

type Aliveness interface {
	SetAlive()
}

type Budgerigar struct {
	searcher *searcher
}

func NewBudgerigar() *Budgerigar {
	return &Budgerigar{
		searcher: newSearcher(),
	}
}

// InternalStorage returns the internal storage interface for adding internal stubs.
// Internal stubs are hidden from user-facing APIs and take precedence in matching.
//
//nolint:ireturn
func (b *Budgerigar) InternalStorage() InternalStubStorage {
	return b.searcher.internalStorage
}

// SetAlive marks internal gripmock health stubs as SERVING.
func (b *Budgerigar) SetAlive() {
	UpdateGripmockHealthStatus(b.searcher.internalStorage, healthgrpc.HealthCheckResponse_SERVING)
}

// PutMany inserts the given Stub values. Assigns UUIDs to stubs without IDs.
func (b *Budgerigar) PutMany(values ...*Stub) []uuid.UUID {
	for _, value := range values {
		if value.ID == uuid.Nil {
			value.ID = uuid.New()
		}
	}

	return b.searcher.upsert(values...)
}

// UpdateMany updates stubs that have non-nil IDs.
func (b *Budgerigar) UpdateMany(values ...*Stub) []uuid.UUID {
	updates := make([]*Stub, 0, len(values))

	for _, value := range values {
		if value.ID != uuid.Nil {
			updates = append(updates, value)
		}
	}

	return b.searcher.upsert(updates...)
}

// DeleteByID deletes the Stub values with the given IDs from the Budgerigar's searcher.
func (b *Budgerigar) DeleteByID(ids ...uuid.UUID) int {
	return b.searcher.del(ids...)
}

// DeleteSession deletes all stubs that belong to the provided session.
// Empty session is treated as global and is not deleted by this method.
func (b *Budgerigar) DeleteSession(session string) int {
	if session == "" {
		return 0
	}

	all := b.searcher.all()
	ids := make([]uuid.UUID, 0, len(all))

	for _, stub := range all {
		if stub.Session == session {
			ids = append(ids, stub.ID)
		}
	}

	if len(ids) == 0 {
		return 0
	}

	return b.searcher.del(ids...)
}

// FindByID retrieves the Stub value associated with the given ID.
func (b *Budgerigar) FindByID(id uuid.UUID) *Stub {
	return b.searcher.findByID(id)
}

// FindByQuery retrieves the Stub value associated with the given Query.
func (b *Budgerigar) FindByQuery(query Query) (*Result, error) {
	return b.searcher.find(query)
}

// FindByQueryBidi retrieves a BidiResult for bidirectional streaming.
func (b *Budgerigar) FindByQueryBidi(query QueryBidi) (*BidiResult, error) {
	return b.searcher.findBidi(query)
}

// FindBy retrieves all Stub values that match the given service and method
// from the Budgerigar's searcher, sorted by priority score in descending order.
func (b *Budgerigar) FindBy(service, method string) ([]*Stub, error) {
	return b.searcher.findBy(service, method)
}

// All returns all Stub values.
func (b *Budgerigar) All() []*Stub {
	stubs := b.searcher.all()
	if stubs == nil {
		return []*Stub{}
	}

	return stubs
}

// Used returns all Stub values that have been used.
func (b *Budgerigar) Used() []*Stub {
	stubs := b.searcher.used()
	if stubs == nil {
		return []*Stub{}
	}

	return stubs
}

// Unused returns all Stub values that have not been used.
func (b *Budgerigar) Unused() []*Stub {
	stubs := b.searcher.unused()
	if stubs == nil {
		return []*Stub{}
	}

	return stubs
}

// Sessions returns sorted non-empty session IDs known by storage.
func (b *Budgerigar) Sessions() []string {
	return b.searcher.sessions()
}

// Clear removes all Stub values.
func (b *Budgerigar) Clear() {
	b.searcher.clear()
}
