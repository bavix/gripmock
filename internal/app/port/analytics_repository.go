package port

import (
	"context"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// AnalyticsRepository defines operations for retrieving per-stub analytics.
type AnalyticsRepository interface {
	// TouchStub updates analytics metrics for a stub after an execution.
	TouchStub(ctx context.Context, stubID string, durationMs int64, wasError bool, sendMsgs int64, dataRes int64, endEvents int64)
	GetByStubID(ctx context.Context, stubID string) (domain.StubAnalytics, bool)
	ListAll(ctx context.Context) []domain.StubAnalytics
}
