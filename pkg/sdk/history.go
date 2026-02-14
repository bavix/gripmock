package sdk

import "github.com/bavix/gripmock/v3/internal/domain/history"

// InMemoryRecorder is an alias for history.MemoryStore for backwards compatibility.
// Use it for Recording and as HistoryReader.
type InMemoryRecorder = history.MemoryStore

// CallRecord is a recorded gRPC call. Re-export for SDK users.
type CallRecord = history.CallRecord
