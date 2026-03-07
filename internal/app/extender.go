package app

import "context"

type instantExtender struct{}

// NewInstantExtender returns an Extender that never blocks.
func NewInstantExtender() *instantExtender {
	return &instantExtender{}
}

func (e *instantExtender) Wait(ctx context.Context) {}
