package protoset

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func fileProto(name, pkg string, deps ...string) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       new(name),
		Package:    new(pkg),
		Syntax:     new("proto3"),
		Dependency: deps,
	}
}

func TestRenameCollidingFilesLeavesFreeNamesAlone(t *testing.T) {
	t.Parallel()

	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{fileProto("rename_free.proto", "rename.free")},
	}

	renameCollidingFiles(t.Context(), "source-a", fds)

	require.Equal(t, "rename_free.proto", fds.GetFile()[0].GetName())
}

func TestRenameCollidingFilesKeepsBothDescriptors(t *testing.T) {
	t.Parallel()

	first := fileProto("rename_clash.proto", "rename.first")
	registerForTest(t, first)

	// Same file name, different content: this is two upstreams shipping their own
	// service.proto, which used to cost the second one all of its services.
	second := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{fileProto("rename_clash.proto", "rename.second")},
	}

	renameCollidingFiles(t.Context(), "proxy-descriptor-set-1", second)

	renamed := second.GetFile()[0].GetName()
	require.NotEqual(t, "rename_clash.proto", renamed)
	require.Contains(t, renamed, "rename_clash.proto")
	require.Contains(t, renamed, "proxy-descriptor-set-1")
}

func TestRenameCollidingFilesRewritesDependencies(t *testing.T) {
	t.Parallel()

	registerForTest(t, fileProto("rename_dep.proto", "rename.dep.first"))

	incoming := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			fileProto("rename_dep.proto", "rename.dep.second"),
			fileProto("rename_user.proto", "rename.user", "rename_dep.proto"),
		},
	}

	renameCollidingFiles(t.Context(), "source-b", incoming)

	renamed := incoming.GetFile()[0].GetName()
	require.NotEqual(t, "rename_dep.proto", renamed)
	// The dependent file must point at the renamed copy, not at the stranger that
	// happens to hold the original name.
	require.Equal(t, []string{renamed}, incoming.GetFile()[1].GetDependency())
}

func TestRenameCollidingFilesIgnoresIdenticalContent(t *testing.T) {
	t.Parallel()

	same := fileProto("rename_same.proto", "rename.same")
	registerForTest(t, same)

	incoming := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fileProto("rename_same.proto", "rename.same")}}

	renameCollidingFiles(t.Context(), "source-c", incoming)

	require.Equal(t, "rename_same.proto", incoming.GetFile()[0].GetName())
}

func registerForTest(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
	t.Helper()

	file, err := protodesc.NewFile(fd, protoregistry.GlobalFiles)
	require.NoError(t, err)

	protoRegistryMu.Lock()
	defer protoRegistryMu.Unlock()

	// The global registry outlives a single test run, so a repeated `-count` pass
	// finds its own file already there.
	_, lookupErr := protoregistry.GlobalFiles.FindFileByPath(fd.GetName())
	if lookupErr == nil {
		return
	}

	require.NoError(t, protoregistry.GlobalFiles.RegisterFile(file))
}

func TestRenameCollidingFilesIsStableAcrossRepeats(t *testing.T) {
	t.Parallel()

	registerForTest(t, fileProto("rename_repeat.proto", "rename.repeat.first"))

	build := func() *descriptorpb.FileDescriptorSet {
		return &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{fileProto("rename_repeat.proto", "rename.repeat.second")},
		}
	}

	first := build()
	renameCollidingFiles(t.Context(), "repeat-source", first)
	registerForTest(t, first.GetFile()[0])

	// Fetching the same upstream again must land on the same name instead of
	// piling up -2, -3 copies of one descriptor.
	second := build()
	renameCollidingFiles(t.Context(), "repeat-source", second)

	require.Equal(t, first.GetFile()[0].GetName(), second.GetFile()[0].GetName())
}

// A vendored well-known file collides by name *and* by symbol. Renaming it would
// only move the clash to symbol registration and take the whole set down with it,
// so such a file is left for the "already registered" path to skip.
func TestRenameCollidingFilesSkipsFilesWhoseSymbolsAreTaken(t *testing.T) {
	t.Parallel()

	registered := fileProto("rename_symbols.proto", "rename.symbols")
	registered.MessageType = []*descriptorpb.DescriptorProto{{Name: new("Shared")}}
	registerForTest(t, registered)

	vendored := fileProto("rename_symbols.proto", "rename.symbols")
	vendored.MessageType = []*descriptorpb.DescriptorProto{
		{Name: new("Shared")},
		{Name: new("Extra")},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{vendored}}

	renameCollidingFiles(t.Context(), "proxy-descriptor-set-0", fds)

	require.Equal(t, "rename_symbols.proto", fds.GetFile()[0].GetName())
}
