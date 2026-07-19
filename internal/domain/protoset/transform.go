package protoset

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
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

		err = RegisterDescriptorSetFiles(ctx, descriptor, fds)
		if err != nil {
			return nil, err
		}

		results = append(results, fds)
	}

	for i, fds := range configure.DescriptorSets() {
		source := "remote-descriptor-set"
		if err := RegisterDescriptorSetFiles(ctx, source, fds); err != nil {
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

		if source.Type == SourceBufBuild || source.Type == SourceReflect || source.Type == SourceProxy {
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
