package app

import "context"

type instantExtender struct{}

func (e *instantExtender) Wait(ctx context.Context) {}

// NewInstantExtender returns an Extender that never blocks.
//

func NewInstantExtender() Extender {
	return &instantExtender{}
}
