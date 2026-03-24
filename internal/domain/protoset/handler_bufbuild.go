package protoset

import (
	"context"
)

type BufBuildHandler struct{}

func (h *BufBuildHandler) CanHandle(raw string) bool {
	_, _, ok := parseBSRModuleRef(raw)

	return ok
}

func (h *BufBuildHandler) Parse(raw string) (*Source, error) {
	module, version, ok := parseBSRModuleRef(raw)
	if !ok {
		module = raw
		version = ""
	}

	return &Source{
		Type:    SourceBufBuild,
		Raw:     raw,
		Module:  module,
		Version: version,
	}, nil
}

func (h *BufBuildHandler) Process(ctx context.Context, source *Source, processor SourceProcessor) error {
	bufProcessor, ok := processor.(interface {
		ProcessBufBuild(ctx context.Context, source *Source) error
	})

	if !ok {
		return nil
	}

	return bufProcessor.ProcessBufBuild(ctx, source)
}
