package proto //nolint:testpackage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	protoPaths := []string{"test1.proto", "test2.proto"}
	imports := []string{"import1", "import2"}

	args := New(protoPaths, imports)

	assert.Equal(t, protoPaths, args.protoPath)
	assert.Equal(t, imports, args.imports)
}

func TestProtoPath(t *testing.T) {
	protoPaths := []string{"test1.proto", "test2.proto"}
	args := New(protoPaths, []string{})

	result := args.ProtoPath()

	assert.Equal(t, protoPaths, result)
}

func TestImports(t *testing.T) {
	imports := []string{"import1", "import2"}
	args := New([]string{}, imports)

	result := args.Imports()

	assert.Equal(t, imports, result)
}

func TestNewWithEmptySlices(t *testing.T) {
	args := New([]string{}, []string{})

	assert.Empty(t, args.ProtoPath())
	assert.Empty(t, args.Imports())
}
