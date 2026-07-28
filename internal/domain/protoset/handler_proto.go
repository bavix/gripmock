package protoset

import (
	"context"
	"strings"
)

type ProtoHandler struct{}

func (h *ProtoHandler) CanHandle(raw string) bool {
	return strings.HasSuffix(raw, ".proto")
}

func (h *ProtoHandler) Parse(raw string) (*Source, error) {
	return &Source{Type: SourceProto, Path: raw, Raw: raw}, nil
}

func (h *ProtoHandler) Process(ctx context.Context, source *Source, processor SourceProcessor) error {
	absPath, err := resolveSourceImportPath(ctx, source, processor)
	if err != nil {
		return err
	}

	processor.AddProtoFile(ctx, absPath)

	return nil
}
