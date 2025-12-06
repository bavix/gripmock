package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

const approxRecordBytes = 1024

// InMemoryHistory stores bounded RPC session history in FIFO order.
type InMemoryHistory struct {
	mu         sync.RWMutex
	maxBytes   int64
	redactKeys map[string]struct{}
	totalBytes int64
	records    []domain.HistoryRecord
}

func NewInMemoryHistory(limitBytes int64, redactKeysCSV string) *InMemoryHistory {
	red := make(map[string]struct{})

	for _, k := range strings.FieldsFunc(strings.TrimSpace(redactKeysCSV), func(r rune) bool { return r == ',' }) {
		k = strings.TrimSpace(k)
		if k != "" {
			red[strings.ToLower(k)] = struct{}{}
		}
	}

	return &InMemoryHistory{maxBytes: limitBytes, redactKeys: red}
}

func (h *InMemoryHistory) Add(_ context.Context, rec domain.HistoryRecord) domain.HistoryRecord {
	if rec.ID == "" {
		rec.ID = uuid.New().String()
	}

	size := approximateSize(rec)

	h.mu.Lock()

	for h.maxBytes > 0 && h.totalBytes+size > h.maxBytes && len(h.records) > 0 {
		removed := h.records[0]
		h.records = h.records[1:]
		h.totalBytes -= approximateSize(removed)
	}

	h.records = append(h.records, rec)
	h.totalBytes += size
	h.mu.Unlock()

	return rec
}

func (h *InMemoryHistory) List(_ context.Context, start, end int) ([]domain.HistoryRecord, int) {
	h.mu.RLock()
	total := len(h.records)

	if start < 0 {
		start = 0
	}

	if end >= total {
		end = total - 1
	}

	if start > end {
		start = end
	}

	out := make([]domain.HistoryRecord, 0, end-start+1)
	if total > 0 && start <= end {
		out = append(out, h.records[start:end+1]...)
	}

	h.mu.RUnlock()

	return out, total
}

func (h *InMemoryHistory) GetByID(_ context.Context, id string) (domain.HistoryRecord, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, r := range h.records {
		if r.ID == id {
			return r, true
		}
	}

	return domain.HistoryRecord{}, false
}

func (h *InMemoryHistory) Clear(_ context.Context) {
	h.mu.Lock()
	h.records = nil
	h.totalBytes = 0
	h.mu.Unlock()
}

func approximateSize(_ domain.HistoryRecord) int64 {
	return approxRecordBytes
}
