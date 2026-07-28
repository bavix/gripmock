package sdk

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// compileProtoFiles compiles .proto source files into a FileDescriptorSet
// (including all transitive dependencies), with protoc's standard well-known-type
// imports available. Each file's directory becomes an import root so sibling
// imports resolve. Returns nil for an empty input.
func compileProtoFiles(ctx context.Context, paths []string) (*descriptorpb.FileDescriptorSet, error) {
	if len(paths) == 0 {
		return &descriptorpb.FileDescriptorSet{}, nil
	}

	rootSeen := make(map[string]bool, len(paths))
	roots := make([]string, 0, len(paths))
	files := make([]string, 0, len(paths))

	for _, p := range paths {
		dir := filepath.Dir(p)
		if !rootSeen[dir] {
			rootSeen[dir] = true
			roots = append(roots, dir)
		}

		files = append(files, filepath.Base(p))
	}

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{ImportPaths: roots}),
	}

	compiled, err := compiler.Compile(ctx, files...)
	if err != nil {
		return nil, fmt.Errorf("gripmock: WithProtoFiles compile failed: %w", err)
	}

	seen := make(map[string]struct{})

	var descriptors []*descriptorpb.FileDescriptorProto

	for _, f := range compiled {
		collectProtoDescriptors(f, seen, &descriptors)
	}

	return &descriptorpb.FileDescriptorSet{File: descriptors}, nil
}

// collectProtoDescriptors gathers a file descriptor and its imports depth-first.
func collectProtoDescriptors(
	file protoreflect.FileDescriptor,
	seen map[string]struct{},
	out *[]*descriptorpb.FileDescriptorProto,
) {
	path := file.Path()
	if _, ok := seen[path]; ok {
		return
	}

	seen[path] = struct{}{}

	imports := file.Imports()
	for i := range imports.Len() {
		collectProtoDescriptors(imports.Get(i).FileDescriptor, seen, out)
	}

	*out = append(*out, protodesc.ToFileDescriptorProto(file))
}
