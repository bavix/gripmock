package descriptors_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

func mustFileDesc(t *testing.T, protoPath string) protoreflect.FileDescriptor { //nolint:ireturn
	t.Helper()

	ctx := t.Context()
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)

	fd := fdsSlice[0].GetFile()[0]
	fileDesc, err := protodesc.NewFile(fd, protoregistry.GlobalFiles)
	require.NoError(t, err)

	return fileDesc
}

func TestRegistryREgisterUnregisterByPath(t *testing.T) {
	t.Parallel()

	reg := descriptors.NewRegistry()
	path := filepath.Join("..", "..", "..", "examples", "projects", "greeter", "service.proto")
	fd := mustFileDesc(t, path)

	reg.Register(fd)
	require.ElementsMatch(t, []string{fd.Path()}, reg.Paths())

	ok := reg.UnregisterByPath(fd.Path())
	require.True(t, ok)
	require.Empty(t, reg.Paths())

	ok = reg.UnregisterByPath(fd.Path())
	require.False(t, ok)
}

func TestRegistryUnregisterByService(t *testing.T) {
	t.Parallel()

	reg := descriptors.NewRegistry()
	path := filepath.Join("..", "..", "..", "examples", "projects", "greeter", "service.proto")
	fd := mustFileDesc(t, path)

	reg.Register(fd)

	n := reg.UnregisterByService("helloworld.Greeter")
	require.Equal(t, 1, n)
	require.Empty(t, reg.Paths())

	n = reg.UnregisterByService("helloworld.Greeter")
	require.Equal(t, 0, n)
}

func TestRegistryRangeFiles(t *testing.T) {
	t.Parallel()

	reg := descriptors.NewRegistry()
	path := filepath.Join("..", "..", "..", "examples", "projects", "greeter", "service.proto")
	fd := mustFileDesc(t, path)

	reg.Register(fd)

	var count int

	reg.RangeFiles(func(protoreflect.FileDescriptor) bool {
		count++

		return true
	})
	require.Equal(t, 1, count)

	reg.UnregisterByPath(fd.Path())

	count = 0

	reg.RangeFiles(func(protoreflect.FileDescriptor) bool {
		count++

		return true
	})
	require.Equal(t, 0, count)
}

func TestRegistryREgisterReplacesExisting(t *testing.T) {
	t.Parallel()

	reg := descriptors.NewRegistry()
	path := filepath.Join("..", "..", "..", "examples", "projects", "greeter", "service.proto")
	fd := mustFileDesc(t, path)

	reg.Register(fd)
	reg.Register(fd)
	require.Len(t, reg.Paths(), 1)
}

func TestRegistryServiceIDs(t *testing.T) {
	t.Parallel()

	reg := descriptors.NewRegistry()
	path := filepath.Join("..", "..", "..", "examples", "projects", "greeter", "service.proto")
	fd := mustFileDesc(t, path)

	reg.Register(fd)

	ids := reg.ServiceIDs()
	require.Contains(t, ids, "helloworld.Greeter")

	reg.UnregisterByService("helloworld.Greeter")
	require.Empty(t, reg.ServiceIDs())
}
