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
//
// Parameters:
// - toggles: The features.Toggles to use.
//
// Returns:
// - A new Budgerigar.
func NewBudgerigar(toggles features.Toggles) *Budgerigar {
	return &Budgerigar{
		searcher: newSearcher(),
		toggles:  toggles,
	}
}

// PutMany inserts the given Stub values into the Budgerigar. If a Stub value
// does not have a key, a new UUID is generated for its key.
//
// Parameters:
// - values: The Stub values to insert.
//
// Returns:
// - []uuid.UUID: The keys of the inserted Stub values.
func (b *Budgerigar) PutMany(values ...*Stub) []uuid.UUID {
	for _, value := range values {
		if value.Key() == uuid.Nil {
			value.ID = uuid.New()
		}
	}

	return b.searcher.upsert(values...)
}

// UpdateMany updates the given Stub values in the Budgerigar. Only Stub values
// with non-nil keys are updated.
//
// Parameters:
// - values: The Stub values to update.
//
// Returns:
// - []uuid.UUID: The keys of the updated values.
func (b *Budgerigar) UpdateMany(values ...*Stub) []uuid.UUID {
	updates := make([]*Stub, 0, len(values))

	for _, value := range values {
		if value.Key() != uuid.Nil {
			updates = append(updates, value)
		}
	}

	return b.searcher.upsert(updates...)
}

// DeleteByID deletes the Stub values with the given IDs from the Budgerigar's searcher.
//
// Parameters:
// - ids: The UUIDs of the Stub values to delete.
//
// Returns:
// - int: The number of Stub values that were successfully deleted.
func (b *Budgerigar) DeleteByID(ids ...uuid.UUID) int {
	return b.searcher.del(ids...)
}

// FindByID retrieves the Stub value associated with the given ID from the Budgerigar's searcher.
//
// Parameters:
// - id: The UUID of the Stub value to retrieve.
//
// Returns:
// - *Stub: The Stub value associated with the given ID, or nil if not found.
func (b *Budgerigar) FindByID(id uuid.UUID) *Stub {
	return b.searcher.findByID(id)
}

// FindByQuery retrieves the Stub value associated with the given Query from the Budgerigar's searcher.
//
// Parameters:
// - query: The Query used to search for a Stub value.
//
// Returns:
// - *Result: The Result containing the found Stub value (if any), or nil.
// - error: An error if the search fails.
func (b *Budgerigar) FindByQuery(query Query) (*Result, error) {
	if b.toggles.Has(MethodTitle) {
		query.Method = cases.
			Title(language.English, cases.NoLower).
			String(query.Method)
	}

	return b.searcher.find(query)
}

// FindByQueryV2 retrieves the Stub value associated with the given QueryV2 from the Budgerigar's searcher.
//
// Parameters:
// - query: The QueryV2 used to search for a Stub value.
//
// Returns:
// - *Result: The Result containing the found Stub value (if any), or nil.
// - error: An error if the search fails.
func (b *Budgerigar) FindByQueryV2(query QueryV2) (*Result, error) {
	if b.toggles.Has(MethodTitle) {
		query.Method = cases.Title(language.English).String(query.Method)
	}

	return b.searcher.findV2(query)
}

// FindByQueryBidi retrieves a BidiResult for bidirectional streaming with the given QueryBidi.
// For bidirectional streaming, each message is treated as a separate unary request.
// The server can respond with multiple messages for each request.
//
// Parameters:
// - query: The QueryBidi used to search for bidirectional streaming stubs.
//
// Returns:
// - *BidiResult: The BidiResult for finding matching stubs for each message.
// - error: An error if the search fails.
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

// All returns all Stub values from the Budgerigar's searcher.
//
// Returns:
// - []*Stub: All Stub values.
func (b *Budgerigar) All() []*Stub {
	return b.searcher.all()
}

// Used returns all Stub values that have been used from the Budgerigar's searcher.
//
// Returns:
// - []*Stub: All used Stub values.
func (b *Budgerigar) Used() []*Stub {
	return b.searcher.used()
}

// Unused returns all Stub values that have not been used from the Budgerigar's searcher.
//
// Returns:
// - []*Stub: All unused Stub values.
func (b *Budgerigar) Unused() []*Stub {
	return b.searcher.unused()
}

// Clear clears all Stub values from the Budgerigar's searcher.
func (b *Budgerigar) Clear() {
	b.searcher.clear()
}
