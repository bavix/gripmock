package protoset

import (
	"context"
	"path/filepath"
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
	absPath, err := filepath.Abs(source.Path)
	if err != nil {
		return err
	}

	processor.AddImportPath(ctx, filepath.Dir(absPath))
	processor.AddProtoFile(ctx, absPath)

	return nil
}
