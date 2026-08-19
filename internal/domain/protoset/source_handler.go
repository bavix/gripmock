package protoset

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
)

var errUnknownSourceScheme = errors.New("unknown source scheme")

// resolveSourceImportPath resolves source.Path to an absolute path and registers
// its directory as an import path, returning the absolute file path.
func resolveSourceImportPath(ctx context.Context, source *Source, processor SourceProcessor) (string, error) {
	absPath, err := filepath.Abs(source.Path)
	if err != nil {
		return "", err
	}

	processor.AddImportPath(ctx, filepath.Dir(absPath))

	return absPath, nil
}

type SourceProcessor interface {
	AddProtoFile(ctx context.Context, filePath string)
	AddDescriptorFile(ctx context.Context, filePath string)
	AddImportPath(ctx context.Context, dir string)
}

type SourceHandler interface {
	CanHandle(raw string) bool
	Parse(raw string) (*Source, error)
	Process(ctx context.Context, source *Source, processor SourceProcessor) error
}

func ParseSource(raw string) (*Source, error) {
	handlers := []SourceHandler{
		&ProxyHandler{},
		&GRPCHandler{},
		&BufBuildHandler{},
		&DescriptorHandler{},
		&ProtoHandler{},
		&DirectoryHandler{},
	}

	for _, handler := range handlers {
		if handler.CanHandle(raw) {
			return handler.Parse(raw)
		}
	}

	if scheme, _, hasScheme := strings.Cut(raw, "://"); hasScheme {
		return nil, errors.Wrapf(errUnknownSourceScheme,
			"%q: use grpc:// or grpcs:// for reflection, "+
				"or {grpc,grpcs}+{proxy,replay,capture}:// for an upstream mode", scheme)
	}

	return &Source{Type: SourceProto, Path: raw, Raw: raw}, nil
}

func ProcessSource(ctx context.Context, source *Source, processor SourceProcessor) error {
	var handler SourceHandler

	switch source.Type {
	case SourceUnknown:
		return nil
	case SourceProxy:
		return nil
	case SourceBufBuild:
		handler = &BufBuildHandler{}
	case SourceReflect:
		handler = &GRPCHandler{}
	case SourceProto:
		handler = &ProtoHandler{}
	case SourceDescriptor:
		handler = &DescriptorHandler{}
	case SourceDirectory:
		handler = &DirectoryHandler{}
	default:
		return nil
	}

	return handler.Process(ctx, source, processor)
}
