package sdk

import "github.com/bavix/gripmock/v3/internal/domain/history"

// InMemoryRecorder is the in-memory call recorder used by the embedded server.
type InMemoryRecorder = history.MemoryStore

// CallRecord is a recorded gRPC call. Re-export for SDK users.
type CallRecord = history.CallRecord
