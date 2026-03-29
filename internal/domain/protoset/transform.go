package protoset

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/pbs"
)

const (
	ProtoExt       = ".proto"
	ProtobufSetExt = ".pb"
	ProtoSetExt    = ".protoset"

	fileTypeProto      = "proto"
	fileTypeDescriptor = "descriptor"
)

//nolint:gochecknoglobals // shared lock is required for GlobalFiles concurrent registration safety
var protoRegistryMu sync.Mutex

var (
	errUnresolvedDescriptorDependencies = errors.New("unresolved descriptor dependencies")
	errDescriptorSymbolConflict         = errors.New("descriptor symbol conflict")
)

type Configure struct {
	imports        []string
	protos         []string
	descriptors    []string
	descriptorSets []*descriptorpb.FileDescriptorSet
}

func (c *Configure) Imports() []string                                 { return c.imports }
func (c *Configure) Protos() []string                                  { return c.protos }
func (c *Configure) Descriptors() []string                             { return c.descriptors }
func (c *Configure) DescriptorSets() []*descriptorpb.FileDescriptorSet { return c.descriptorSets }

func createDescriptorSet(ctx context.Context, configure *Configure) (*descriptorpb.FileDescriptorSet, error) {
	failbackResolver, err := pbs.NewResolver()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fallback resolver")
	}

	compiler := protocompile.Compiler{
		Resolver: protocompile.CompositeResolver{
			&protocompile.SourceResolver{
				ImportPaths: configure.Imports(),
			},
			failbackResolver,
		},
	}

	files, err := compiler.Compile(ctx, configure.Protos()...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile descriptors")
	}

	fds := &descriptorpb.FileDescriptorSet{
		File: make([]*descriptorpb.FileDescriptorProto, len(files)),
	}

	for i, file := range files {
		fdp := protodesc.ToFileDescriptorProto(file)
		fds.File[i] = fdp

		err = registerGlobalFileOnce(ctx, fdp.GetName(), file.Path(), file)
		if err != nil {
			return nil, err
		}
	}

	return fds, nil
}

func compile(ctx context.Context, configure *Configure) ([]*descriptorpb.FileDescriptorSet, error) {
	capacity := len(configure.Descriptors()) + len(configure.DescriptorSets())
	if len(configure.Protos()) > 0 {
		capacity++
	}

	results := make([]*descriptorpb.FileDescriptorSet, 0, capacity)

	for _, descriptor := range configure.Descriptors() {
		descriptorBytes, err := os.ReadFile(descriptor) //nolint:gosec
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read descriptor: %s", descriptor)
		}

		fds := &descriptorpb.FileDescriptorSet{}

		err = proto.Unmarshal(descriptorBytes, fds)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal descriptor: %s", descriptor)
		}

		err = registerDescriptorSetFiles(ctx, descriptor, fds)
		if err != nil {
			return nil, err
		}

		results = append(results, fds)
	}

	for i, fds := range configure.DescriptorSets() {
		source := "remote-descriptor-set"
		if err := registerDescriptorSetFiles(ctx, source, fds); err != nil {
			return nil, errors.Wrapf(err, "failed to register in-memory descriptor set: %d", i)
		}

		results = append(results, fds)
	}

	if len(configure.Protos()) > 0 {
		fds, err := createDescriptorSet(ctx, configure)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create descriptor set")
		}

		results = append(results, fds)
	}

	return results, nil
}

//nolint:funlen,wsl_v5
func registerDescriptorSetFiles(
	ctx context.Context,
	descriptorPath string,
	fds *descriptorpb.FileDescriptorSet,
) error {
	pending := slices.Clone(fds.GetFile())

	for len(pending) > 0 {
		next := make([]*descriptorpb.FileDescriptorProto, 0, len(pending))
		progress := false
		var lastErr error

		for _, fd := range pending {
			protoRegistryMu.Lock()

			if value, _ := protoregistry.GlobalFiles.FindFileByPath(fd.GetName()); value != nil {
				protoRegistryMu.Unlock()

				zerolog.Ctx(ctx).Warn().
					Str("name", fd.GetName()).
					Str("path", descriptorPath).
					Msg("File already registered")

				progress = true

				continue
			}

			fileDesc, err := protodesc.NewFile(fd, protoregistry.GlobalFiles)
			if err != nil {
				protoRegistryMu.Unlock()
				lastErr = err
				next = append(next, fd)

				continue
			}

			conflict, registerErr := registerGlobalFile(fileDesc)
			protoRegistryMu.Unlock()

			if conflict {
				zerolog.Ctx(ctx).Warn().
					Str("name", fd.GetName()).
					Str("path", descriptorPath).
					Msg("Descriptor conflicts with existing symbols; skipping")

				progress = true

				continue
			}

			if registerErr != nil {
				lastErr = registerErr
				next = append(next, fd)

				continue
			}

			progress = true
		}

		if len(next) == 0 {
			return nil
		}

		if !progress {
			return errors.Wrapf(lastErr, "%w: failed to register file %s", errUnresolvedDescriptorDependencies, descriptorPath)
		}

		pending = next
	}

	return nil
}

func registerGlobalFile(file protoreflect.FileDescriptor) (bool, error) {
	var (
		conflict bool
		err      error
	)

	defer func() {
		if recovered := recover(); recovered != nil {
			conflict = true
			err = errors.Wrapf(errDescriptorSymbolConflict, "%v", recovered)
		}
	}()

	err = protoregistry.GlobalFiles.RegisterFile(file)

	return conflict, err
}

func registerGlobalFileOnce(
	ctx context.Context,
	fileName string,
	filePath string,
	file protoreflect.FileDescriptor,
) error {
	protoRegistryMu.Lock()
	defer protoRegistryMu.Unlock()

	if value, _ := protoregistry.GlobalFiles.FindFileByPath(fileName); value != nil {
		zerolog.Ctx(ctx).Warn().
			Str("name", fileName).
			Str("path", filePath).
			Msg("File already registered")

		return nil
	}

	err := protoregistry.GlobalFiles.RegisterFile(file)
	if err != nil {
		return errors.Wrapf(err, "failed to register file %s", filePath)
	}

	return nil
}

func newConfigure(ctx context.Context, imports []string, paths []string, remoteClient RemoteClient) (*Configure, error) {
	p := newProcessor(imports, remoteClient)

	err := p.process(ctx, paths)
	if err != nil {
		_ = p.Cleanup()

		return nil, errors.Wrap(err, "failed to create configuration")
	}

	return p.result(), nil
}

func findMinimalPaths(paths []string) []string {
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) < len(paths[j])
	})

	var result []string

	for _, path := range paths {
		isSubPath := false

		for _, existing := range result {
			rel, err := filepath.Rel(existing, path)
			if err != nil {
				continue
			}

			if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
				isSubPath = true

				break
			}
		}

		if !isSubPath {
			result = append(result, path)
		}
	}

	return result
}

func Build(
	ctx context.Context,
	imports []string,
	paths []string,
	remoteClient RemoteClient,
) ([]*descriptorpb.FileDescriptorSet, error) {
	var err error

	for i, importPath := range imports {
		imports[i], err = filepath.Abs(importPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve import path: %s", importPath)
		}
	}

	for i, path := range paths {
		source, err := ParseSource(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse source: %s", path)
		}

		if source.Type == SourceBufBuild || source.Type == SourceReflect {
			continue
		}

		paths[i], err = filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve path: %s", path)
		}
	}

	configure, err := newConfigure(ctx, lo.Uniq(findMinimalPaths(imports)), lo.Uniq(paths), remoteClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create configuration")
	}

	return compile(ctx, configure)
}

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
func (p *processor) processDirectory(ctx context.Context, absPath string) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("directory", absPath).Msg("Walking directory")

	p.addImport(ctx, absPath)

	return errors.Wrapf(filepath.Walk(absPath, func(pth string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return errors.Wrapf(err, "failed to access path %s", pth)
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(pth)
		logger := logger.With().
			Str("file", pth).
			Str("extension", ext).
			Logger()

		switch {
		case slices.Contains(p.allowedProtoExts, ext):
			logger.Debug().Msg("Found proto file")
			p.AddProtoFile(ctx, pth)
		case slices.Contains(p.allowedDescExts, ext):
			logger.Debug().Msg("Found descriptor file")
			p.AddDescriptorFile(ctx, pth)
		default:
			logger.Debug().Msg("Skipping unsupported file type")
		}

		return nil
	}), "failed to walk directory %s", absPath)
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
