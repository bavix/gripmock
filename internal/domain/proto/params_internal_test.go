package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	protoPaths := []string{"test1.proto", "test2.proto"}
	imports := []string{"import1", "import2"}

	args := New(protoPaths, imports)

	require.Equal(t, protoPaths, args.protoPath)
	require.Equal(t, imports, args.imports)
}

func TestProtoPath(t *testing.T) {
	t.Parallel()

	protoPaths := []string{"test1.proto", "test2.proto"}
	args := New(protoPaths, []string{})

	result := args.ProtoPath()

	require.Equal(t, protoPaths, result)
}

func TestImports(t *testing.T) {
	t.Parallel()

	imports := []string{"import1", "import2"}
	args := New([]string{}, imports)

	result := args.Imports()

	require.Equal(t, imports, result)
}

func TestNewWithEmptySlices(t *testing.T) {
	t.Parallel()

	args := New([]string{}, []string{})

	require.Empty(t, args.ProtoPath())
	require.Empty(t, args.Imports())
}
