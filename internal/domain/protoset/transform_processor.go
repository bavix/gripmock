package protoset

import (
	"context"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/descriptorpb"
)

type processor struct {
	imports          []string
	protos           []string
	descriptors      []string
	descriptorSets   []*descriptorpb.FileDescriptorSet
	remoteClient     RemoteClient
	seenDirs         map[string]bool
	seenFiles        map[string]bool
	allowedProtoExts []string
	allowedDescExts  []string
}

func newProcessor(initialImports []string, remoteClient RemoteClient) *processor {
	return &processor{
		imports:      initialImports,
		remoteClient: remoteClient,
		seenDirs:     make(map[string]bool),
		seenFiles:    make(map[string]bool),
		allowedProtoExts: []string{
			ProtoExt,
		},
		allowedDescExts: []string{
			ProtobufSetExt,
			ProtoSetExt,
		},
	}
}

func (p *processor) Cleanup() error {
	return nil
}

//nolint:funcorder
func (p *processor) process(ctx context.Context, paths []string) error {
	logger := zerolog.Ctx(ctx)

	for _, path := range paths {
		select {
		case <-ctx.Done():
			return ctx.Err() //nolint:wrapcheck
		default:
		}

		logger.Debug().Str("path", path).Msg("Processing path")

		source, err := ParseSource(path)
		if err != nil {
			return errors.Wrapf(err, "failed to parse source: %s", path)
		}

		if source.Type == SourceBufBuild {
			logger.Info().Str("module", source.Module).Str("version", source.Version).Msg("Processing buf.build module")
		}

		if source.Type == SourceReflect {
			logger.Info().
				Str("address", source.ReflectAddress).
				Bool("tls", source.ReflectTLS).
				Msg("Processing gRPC remote source")
		}

		err = ProcessSource(ctx, source, p)
		if err != nil {
			return errors.Wrapf(err, "failed to process source: %s", source.Raw)
		}
	}

	return nil
}

//nolint:funcorder
func (p *processor) addImport(ctx context.Context, dir string) {
	var (
		dirAbs string
		err    error
	)

	dirAbs, err = filepath.Abs(dir)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Str("dir", dir).Msg("Failed to resolve absolute path for import")

		return
	}

	if !p.seenDirs[dirAbs] {
		beforeLen := len(p.imports)

		p.imports = findMinimalPaths(append(p.imports, dirAbs))
		p.seenDirs[dirAbs] = true

		if len(p.imports) > beforeLen {
			zerolog.Ctx(ctx).Debug().Str("import", dirAbs).Msg("Added import path")
		}
	}
}

func (p *processor) AddProtoFile(ctx context.Context, filePath string) {
	p.addFile(ctx, filePath, fileTypeProto)
}

func (p *processor) AddDescriptorFile(ctx context.Context, filePath string) {
	p.addFile(ctx, filePath, fileTypeDescriptor)
}

func (p *processor) AddImportPath(ctx context.Context, dir string) {
	p.addImport(ctx, dir)
}

func (p *processor) ProcessBufBuild(ctx context.Context, source *Source) error {
	if p.remoteClient == nil {
		return nil
	}

	fds, err := p.remoteClient.FetchDescriptorSet(ctx, source)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch from buf.build: %s", source.Raw)
	}

	p.descriptorSets = append(p.descriptorSets, fds)

	return nil
}

func (p *processor) ProcessReflect(ctx context.Context, source *Source) error {
	if p.remoteClient == nil {
		return nil
	}

	fds, err := p.remoteClient.FetchDescriptorSet(ctx, source)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch from gRPC reflection: %s", source.Raw)
	}

	p.descriptorSets = append(p.descriptorSets, fds)

	return nil
}

func findPathByImports(filePath string, imports []string) (string, string) {
	filePath = filepath.ToSlash(filePath)

	sort.Slice(imports, func(i, j int) bool {
		return len(imports[i]) > len(imports[j])
	})

	for _, imp := range imports {
		impPath := filepath.ToSlash(imp)

		if !strings.HasSuffix(impPath, "/") {
			impPath += "/"
		}

		if strings.HasPrefix(filePath, impPath) {
			relPath := filePath[len(impPath):]

			return filepath.FromSlash(imp), filepath.FromSlash(relPath)
		}
	}

	return "", filepath.Base(filePath)
}

func (p *processor) addFile(ctx context.Context, filePath, fileType string) {
	var (
		fileAbs string
		err     error
	)

	fileAbs, err = filepath.Abs(filePath)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Str("file", filePath).Msg("Failed to resolve absolute path")

		return
	}

	if p.seenFiles[fileAbs] {
		zerolog.Ctx(ctx).Debug().Msg("File already processed")

		return
	}

	baseDir, _ := findPathByImports(fileAbs, p.imports)

	relPath, err := filepath.Rel(baseDir, fileAbs)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Str("file", fileAbs).Str("base_dir", baseDir).Msg("Failed to get relative path")

		return
	}

	switch fileType {
	case fileTypeProto:
		p.protos = append(p.protos, relPath)
	case fileTypeDescriptor:
		p.descriptors = append(p.descriptors, fileAbs)
	default:
		zerolog.Ctx(ctx).Error().Str("file_type", fileType).Msg("Unknown file type encountered")

		return
	}

	p.seenFiles[fileAbs] = true

	zerolog.Ctx(ctx).Debug().Str("type", fileType).Msg("File added successfully")
}

func (p *processor) result() *Configure {
	return &Configure{
		imports:        lo.Uniq(p.imports),
		protos:         lo.Uniq(p.protos),
		descriptors:    lo.Uniq(p.descriptors),
		descriptorSets: slices.Clone(p.descriptorSets),
	}
}
