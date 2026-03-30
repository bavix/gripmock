package fdstest

import (
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func DescriptorSetFromFile(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	fds := &descriptorpb.FileDescriptorSet{}
	seen := map[string]struct{}{}
	appendFileRecursive(fds, seen, root)

	return fds
}

func appendFileRecursive(
	fds *descriptorpb.FileDescriptorSet,
	seen map[string]struct{},
	fd protoreflect.FileDescriptor,
) {
	name := fd.Path()
	if _, ok := seen[name]; ok {
		return
	}
	seen[name] = struct{}{}

	imports := fd.Imports()
	for i := 0; i < imports.Len(); i++ {
		appendFileRecursive(fds, seen, imports.Get(i).FileDescriptor)
	}

	fds.File = append(fds.File, protodesc.ToFileDescriptorProto(fd))
}
