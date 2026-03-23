package protoset

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type DirectoryHandler struct{}

func (h *DirectoryHandler) CanHandle(raw string) bool {
	if strings.HasSuffix(raw, ".proto") || strings.HasSuffix(raw, ".pb") || strings.HasSuffix(raw, ".protoset") {
		return false
	}

	info, err := os.Stat(raw)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func (h *DirectoryHandler) Parse(raw string) (*Source, error) {
	return &Source{Type: SourceDirectory, Path: raw, Raw: raw}, nil
}

func (h *DirectoryHandler) Process(ctx context.Context, source *Source, processor SourceProcessor) error {
	processor.AddImportPath(ctx, source.Path)

	return filepath.Walk(source.Path, func(pth string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(pth)
		absPath, _ := filepath.Abs(pth)

		switch ext {
		case ".proto":
			processor.AddProtoFile(ctx, absPath)
		case ".pb", ".protoset":
			processor.AddDescriptorFile(ctx, absPath)
		}

		return nil
	})
}
