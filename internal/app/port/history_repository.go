package port

import (
	"context"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// HistoryRepository defines operations for session history storage.
type HistoryRepository interface {
	Add(ctx context.Context, rec domain.HistoryRecord) domain.HistoryRecord
	List(ctx context.Context, start, end int) ([]domain.HistoryRecord, int)
	GetByID(ctx context.Context, id string) (domain.HistoryRecord, bool)
	Clear(ctx context.Context)
}
