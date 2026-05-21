package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	protoPaths := []string{"test1.proto", "test2.proto"}
	imports := []string{"import1", "import2"}
	sources := []string{"source1.proto"}

	args := New(protoPaths, imports, sources)

	require.Equal(t, protoPaths, args.protoPath)
	require.Equal(t, imports, args.imports)
	require.Equal(t, sources, args.sources)
}

func TestProtoPath(t *testing.T) {
	t.Parallel()

	protoPaths := []string{"test1.proto", "test2.proto"}
	args := New(protoPaths, []string{}, []string{})

	result := args.ProtoPath()

	require.Equal(t, protoPaths, result)
}

func TestImports(t *testing.T) {
	t.Parallel()

	imports := []string{"import1", "import2"}
	args := New([]string{}, imports, []string{})

	result := args.Imports()

	require.Equal(t, imports, result)
}

func TestSources(t *testing.T) {
	t.Parallel()

	sources := []string{"source1.proto", "source2.proto"}
	args := New([]string{}, []string{}, sources)

	result := args.Sources()

	require.Equal(t, sources, result)
}

func TestNewWithEmptySlices(t *testing.T) {
	t.Parallel()

	args := New([]string{}, []string{}, []string{})

	require.Empty(t, args.ProtoPath())
	require.Empty(t, args.Imports())
	require.Empty(t, args.Sources())
}
