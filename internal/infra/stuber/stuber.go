package stuber

import (
	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/bavix/features"
)

// MethodTitle is a feature flag for using title casing in the method field
// of a Query struct.
const MethodTitle features.Flag = iota

// Budgerigar is the main struct for the stuber package. It contains a
// searcher and toggles.
type Budgerigar struct {
	searcher *searcher
	toggles  features.Toggles
}

// NewBudgerigar creates a new Budgerigar with the given features.Toggles.
func NewBudgerigar(toggles features.Toggles) *Budgerigar {
	return &Budgerigar{
		searcher: newSearcher(),
		toggles:  toggles,
	}
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

// FindByID retrieves the Stub value associated with the given ID.
func (b *Budgerigar) FindByID(id uuid.UUID) *Stub {
	return b.searcher.findByID(id)
}

// FindByQuery retrieves the Stub value associated with the given Query.
func (b *Budgerigar) FindByQuery(query Query) (*Result, error) {
	if b.toggles.Has(MethodTitle) {
		query.Method = cases.
			Title(language.English, cases.NoLower).
			String(query.Method)
	}

	return b.searcher.find(query)
}

// FindByQueryBidi retrieves a BidiResult for bidirectional streaming.
func (b *Budgerigar) FindByQueryBidi(query QueryBidi) (*BidiResult, error) {
	if b.toggles.Has(MethodTitle) {
		query.Method = cases.Title(language.English).String(query.Method)
	}

	return b.searcher.findBidi(query)
}

// FindBy retrieves all Stub values that match the given service and method
// from the Budgerigar's searcher, sorted by priority score in descending order.
func (b *Budgerigar) FindBy(service, method string) ([]*Stub, error) {
	return b.searcher.findBy(service, method)
}

// All returns all Stub values.
func (b *Budgerigar) All() []*Stub {
	return b.searcher.all()
}

// Used returns all Stub values that have been used.
func (b *Budgerigar) Used() []*Stub {
	return b.searcher.used()
}

// Unused returns all Stub values that have not been used.
func (b *Budgerigar) Unused() []*Stub {
	return b.searcher.unused()
}

// Clear removes all Stub values.
func (b *Budgerigar) Clear() {
	b.searcher.clear()
}
