package yaml2json

import (
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

type engine struct{}

func newEngine(_ plugins.Registry) *engine {
	return &engine{}
}

func (e *engine) Execute(name string, data []byte) ([]byte, error) {
	_ = name

	// Do not pre-process templates; pass through as-is.
	// Optimization: handled upstream (Load path can skip template stage if no markers).
	return data, nil
}
