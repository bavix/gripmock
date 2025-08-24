package memory

import (
	"context"
	"sync"
	"time"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// InMemoryAnalytics stores per-stub analytics counters.
type InMemoryAnalytics struct {
	mu    sync.RWMutex
	items map[string]*domain.StubAnalytics
}

func NewInMemoryAnalytics() *InMemoryAnalytics {
	return &InMemoryAnalytics{items: make(map[string]*domain.StubAnalytics)}
}

func (a *InMemoryAnalytics) TouchStub(
	_ context.Context,
	stubID string,
	durationMs int64,
	wasError bool,
	sendMsgs int64,
	dataRes int64,
	endEvents int64,
) {
	now := time.Now()

	a.mu.Lock()

	item, ok := a.items[stubID]
	if !ok {
		item = &domain.StubAnalytics{StubID: stubID}
		a.items[stubID] = item
	}

	item.UsedCount++
	if item.FirstUsedAt == nil {
		item.FirstUsedAt = &now
	}

	item.LastUsedAt = &now
	item.TotalSendMessages += sendMsgs
	item.TotalDataResponses += dataRes

	item.StreamEndEvents += endEvents
	if wasError {
		item.ErrorCount++
	}

	item.TotalDurationMilliseconds += durationMs
	if item.UsedCount > 0 {
		item.AverageDurationMilliseconds = float64(item.TotalDurationMilliseconds) / float64(item.UsedCount)
	}

	a.mu.Unlock()
}

func (a *InMemoryAnalytics) GetByStubID(_ context.Context, stubID string) (domain.StubAnalytics, bool) {
	a.mu.RLock()
	item, ok := a.items[stubID]
	a.mu.RUnlock()

	if !ok {
		return domain.StubAnalytics{}, false
	}

	return *item, true
}

func (a *InMemoryAnalytics) ListAll(_ context.Context) []domain.StubAnalytics {
	a.mu.RLock()

	out := make([]domain.StubAnalytics, 0, len(a.items))
	for _, it := range a.items {
		out = append(out, *it)
	}

	a.mu.RUnlock()

	return out
}
