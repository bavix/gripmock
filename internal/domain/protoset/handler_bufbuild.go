package protoset

import (
	"context"
	"strings"
)

type BufBuildHandler struct{}

func (h *BufBuildHandler) CanHandle(raw string) bool {
	return strings.HasPrefix(raw, "buf.build/")
}

func (h *BufBuildHandler) Parse(raw string) (*Source, error) {
	module := raw
	version := ""

	if before, after, ok := strings.Cut(raw, "@"); ok {
		module, version = before, after
	} else if before, after, ok := strings.Cut(raw, ":"); ok {
		module, version = before, after
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
