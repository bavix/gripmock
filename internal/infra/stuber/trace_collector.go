package stuber

import "github.com/google/uuid"

// traceCollector is an internal hook interface used by search workflow
// to report stage transitions for inspect/debug use-cases.
//
// It is intentionally package-private while the design is evolving.
type traceCollector interface {
	addStage(name string, before, after int)
	setFallbackToMethod(v bool)
	setMatchedStubID(id *uuid.UUID)
}
