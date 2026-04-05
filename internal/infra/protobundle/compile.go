package protobundle

import (
	"context"
	"sort"

	"github.com/bufbuild/protocompile"
	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// CompileParams configures the protocompile compilation.
type CompileParams struct {
	Roots []string // import roots for resolver
	Files []string // relative paths to compile
}

// Compile produces a FileDescriptorSet from discovered proto files.
// The returned descriptor set contains all compiled files and their transitive dependencies,
// sorted by file name for deterministic output.
func Compile(ctx context.Context, params CompileParams) (*descriptorpb.FileDescriptorSet, error) {
	if len(params.Files) == 0 {
		return nil, errors.New("no files to compile")
	}

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: params.Roots,
		},
	}

	compiled, err := compiler.Compile(ctx, params.Files...)
	if err != nil {
		return nil, errors.Wrap(err, "protocompile failed")
	}

	// Collect all file descriptors including transitive dependencies.
	seen := make(map[string]struct{})

	var descriptors []*descriptorpb.FileDescriptorProto

	for _, file := range compiled {
		collectDescriptors(file, seen, &descriptors)
	}

	// Sort by file name for deterministic output.
	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].GetName() < descriptors[j].GetName()
	})

	return &descriptorpb.FileDescriptorSet{File: descriptors}, nil
}

// collectDescriptors recursively collects a file descriptor and its imports.
func collectDescriptors(
	file protoreflect.FileDescriptor,
	seen map[string]struct{},
	result *[]*descriptorpb.FileDescriptorProto,
) {
	path := file.Path()
	if _, ok := seen[path]; ok {
		return
	}

	seen[path] = struct{}{}

	// Collect dependencies first (depth-first).
	imports := file.Imports()
	for i := range imports.Len() {
		collectDescriptors(imports.Get(i).FileDescriptor, seen, result)
	}

	*result = append(*result, protodesc.ToFileDescriptorProto(file))
}
