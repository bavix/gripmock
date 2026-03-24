package protoset

import (
	"context"
	"path/filepath"
	"strings"
)

type DescriptorHandler struct{}

func (h *DescriptorHandler) CanHandle(raw string) bool {
	return strings.HasSuffix(raw, ".pb") || strings.HasSuffix(raw, ".protoset")
}

func (h *DescriptorHandler) Parse(raw string) (*Source, error) {
	return &Source{Type: SourceDescriptor, Path: raw, Raw: raw}, nil
}

func (h *DescriptorHandler) Process(ctx context.Context, source *Source, processor SourceProcessor) error {
	absPath, err := filepath.Abs(source.Path)
	if err != nil {
		return err
	}

	processor.AddImportPath(ctx, filepath.Dir(absPath))
	processor.AddDescriptorFile(ctx, absPath)

	return nil
}
