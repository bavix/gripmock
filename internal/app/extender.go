package app

import "context"

type instantExtender struct{}

// NewInstantExtender returns an Extender that never blocks.
func NewInstantExtender() *instantExtender {
	return &instantExtender{}
}

func (e *instantExtender) Wait(_ context.Context) {
	// No-op: the instant extender is always ready, so there is nothing to wait for.
}
