package port

import (
	"context"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// StubFilter defines filtering options for listing stubs.
type StubFilter struct {
	Service string   `json:"service,omitempty"`
	Method  string   `json:"method,omitempty"`
	Used    *bool    `json:"used,omitempty"`
	Query   string   `json:"q,omitempty"`
	IDs     []string `json:"ids,omitempty"`
}

// SortOption represents sorting by a field and direction.
type SortOption struct {
	Field     string
	Direction string // "ASC" or "DESC"
}

// RangeOption represents inclusive paging range [Start, End].
type RangeOption struct {
	Start int
	End   int
}

// StubRepository defines operations for managing v4 stubs.
type StubRepository interface {
	Create(ctx context.Context, stub domain.Stub) (domain.Stub, error)
	Update(ctx context.Context, id string, stub domain.Stub) (domain.Stub, error)
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	GetByID(ctx context.Context, id string) (domain.Stub, bool)
	List(ctx context.Context, filter StubFilter, sort SortOption, r RangeOption) (items []domain.Stub, total int)
}
