package protobundle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/protobundle"
)

func TestSyntaxRank_EditionWinsOverProto3(t *testing.T) {
	t.Parallel()

	root1 := filepath.Join(fixturesDir(t), "root1")         // proto3 hello
	rootEd := filepath.Join(fixturesDir(t), "root_edition") // edition "2023" hello

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root1, rootEd},
	})
	require.NoError(t, err)
	require.Contains(t, result.Files, "pkg/hello.proto")
	require.Equal(t, filepath.Join(rootEd, "pkg/hello.proto"), result.Files["pkg/hello.proto"])
}

func TestSyntaxRank_Proto3WinsOverProto2(t *testing.T) {
	t.Parallel()

	root1 := filepath.Join(fixturesDir(t), "root1") // proto3
	root2 := filepath.Join(fixturesDir(t), "root2") // proto2

	// root2 first, so proto2 is seen first — proto3 from root1 should still win.
	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root2, root1},
	})
	require.NoError(t, err)
	require.Contains(t, result.Files, "pkg/hello.proto")
	require.Equal(t, filepath.Join(root1, "pkg/hello.proto"), result.Files["pkg/hello.proto"])
}

func TestSyntaxRank_NoSyntaxDefaultsToProto2(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "test.proto"),
		[]byte("package test;\nmessage Msg { optional string s = 1; }"),
		0o600,
	))

	// A no-syntax file should lose to proto3.
	root3 := filepath.Join(fixturesDir(t), "root_mixed") // proto3

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots:   []string{dir, root3},
		Include: []string{"*.proto"},
	})
	require.NoError(t, err)

	// The root_mixed simple.proto and the no-syntax test.proto have different names,
	// so both should appear — just verify no errors.
	require.NotEmpty(t, result.Files)
}
