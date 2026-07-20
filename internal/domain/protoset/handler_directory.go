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

	// Track which proto basenames we've seen to skip duplicate .pb/.protoset
	seenProto := make(map[string]bool)

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
			seenProto[absPath] = true
			processor.AddProtoFile(ctx, absPath)
		case ".pb", ".protoset":
			// Skip .pb/.protoset if a .proto with the same base name exists in the same dir
			protoPath := absPath[:len(absPath)-len(ext)] + ".proto"
			if seenProto[protoPath] {
				return nil
			}
			// Also check on disk (in case .proto was already registered via walk order
			// or in a different handler earlier)
			if _, statErr := os.Stat(protoPath); statErr == nil {
				return nil
			}

			processor.AddDescriptorFile(ctx, absPath)
		}

		return nil
	})
}
