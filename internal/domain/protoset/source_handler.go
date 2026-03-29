package protoset

import "context"

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

	return &Source{Type: SourceProto, Path: raw, Raw: raw}, nil
}

func ProcessSource(ctx context.Context, source *Source, processor SourceProcessor) error {
	var handler SourceHandler

	switch source.Type {
	case SourceUnknown:
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
