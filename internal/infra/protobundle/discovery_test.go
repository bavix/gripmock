package protobundle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/protobundle"
)

func fixturesDir(t *testing.T) string {
	t.Helper()

	// Walk up from the test file to find the project root (where go.mod lives),
	// then return third_party/protobundle.
	dir, err := filepath.Abs(".")
	require.NoError(t, err)

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return filepath.Join(dir, "third_party", "protobundle")
		}

		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "go.mod not found")

		dir = parent
	}
}

func TestDiscoverSingleRoot(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root1")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root},
	})
	require.NoError(t, err)
	require.Len(t, result.Files, 2)
	require.Contains(t, result.Files, "pkg/hello.proto")
	require.Contains(t, result.Files, "pkg/world.proto")
	require.Empty(t, result.Skipped)
}

func TestDiscoverMultipleRootsProto3WinsOverProto2(t *testing.T) {
	t.Parallel()

	root1 := filepath.Join(fixturesDir(t), "root1") // proto3 hello
	root2 := filepath.Join(fixturesDir(t), "root2") // proto2 hello + extra

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root1, root2},
	})
	require.NoError(t, err)

	// hello.proto from root1 (proto3) should win over root2 (proto2).
	require.Contains(t, result.Files, "pkg/hello.proto")
	require.Equal(t, filepath.Join(root1, "pkg/hello.proto"), result.Files["pkg/hello.proto"])

	// extra.proto only in root2.
	require.Contains(t, result.Files, "pkg/extra.proto")

	// world.proto only in root1.
	require.Contains(t, result.Files, "pkg/world.proto")

	require.Len(t, result.Files, 3)
	require.Len(t, result.Skipped, 1) // root2 hello was skipped
}

func TestDiscoverEditionWinsOverProto3(t *testing.T) {
	t.Parallel()

	root1 := filepath.Join(fixturesDir(t), "root1")              // proto3 hello
	rootEdition := filepath.Join(fixturesDir(t), "root_edition") // edition "2023" hello

	// root1 first, then root_edition — edition should still win.
	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root1, rootEdition},
	})
	require.NoError(t, err)

	require.Contains(t, result.Files, "pkg/hello.proto")
	require.Equal(t, filepath.Join(rootEdition, "pkg/hello.proto"), result.Files["pkg/hello.proto"])
	require.Len(t, result.Skipped, 1)
}

func TestDiscoverExcludePattern(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root1")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots:   []string{root},
		Exclude: []string{"pkg/world.proto"},
	})
	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	require.Contains(t, result.Files, "pkg/hello.proto")
}

func TestDiscoverExcludeGlobPattern(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root1")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots:   []string{root},
		Exclude: []string{"pkg/**"},
	})
	require.NoError(t, err)
	require.Empty(t, result.Files)
}

func TestDiscoverIncludePattern(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root1")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots:   []string{root},
		Include: []string{"pkg/hello.proto"},
	})
	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	require.Contains(t, result.Files, "pkg/hello.proto")
}

func TestDiscoverNoRoots(t *testing.T) {
	t.Parallel()

	_, err := protobundle.Discover(protobundle.DiscoverParams{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one root")
}

func TestDiscoverNonExistentRoot(t *testing.T) {
	t.Parallel()

	_, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{"/nonexistent/path/to/root"},
	})
	require.Error(t, err)
}

func TestDiscoverIgnoresNonProtoFiles(t *testing.T) {
	t.Parallel()

	// Create a temp dir with mixed files.
	dir := t.TempDir()

	protoContent := `syntax = "proto3";
package test;
message Msg { string s = 1; }
`

	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.go"), []byte("package test"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "msg.proto"), []byte(protoContent), 0o600))

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{dir},
	})
	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	require.Contains(t, result.Files, "msg.proto")
}

func TestDiscoverUnsupportedEditionSkipped(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_edition2024")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root},
	})
	require.NoError(t, err)

	require.Empty(t, result.UnsupportedEdition)
	require.Len(t, result.Files, 2)
}

func TestDiscoverEdition2024MixedWithSupported(t *testing.T) {
	t.Parallel()

	root1 := filepath.Join(fixturesDir(t), "root1")
	root2024 := filepath.Join(fixturesDir(t), "root_edition2024")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root1, root2024},
	})
	require.NoError(t, err)

	require.Len(t, result.Files, 3)
	require.Contains(t, result.Files, "pkg/hello.proto")
	require.Contains(t, result.Files, "pkg/world.proto")
	require.Contains(t, result.Files, "pkg/extra.proto")

	require.Empty(t, result.UnsupportedEdition)
}

func TestDiscoverCustomMaxEdition(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_edition2024")

	// With MaxEdition=2024, edition "2024" files should be accepted.
	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots:      []string{root},
		MaxEdition: 2024,
	})
	require.NoError(t, err)
	require.Len(t, result.Files, 2)
	require.Empty(t, result.UnsupportedEdition)
}

func TestDiscoverNonNumericEditionSkipped(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_unstable")

	// edition = "UNSTABLE" is non-numeric and should be treated as unsupported.
	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root},
	})
	require.NoError(t, err)
	require.Empty(t, result.Files)
	require.Len(t, result.UnsupportedEdition, 1)
	require.Contains(t, result.UnsupportedEdition, "pkg/hello.proto")
}

func TestDiscoverSorted(t *testing.T) {
	t.Parallel()

	root1 := filepath.Join(fixturesDir(t), "root1")
	root2 := filepath.Join(fixturesDir(t), "root2")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots: []string{root1, root2},
	})
	require.NoError(t, err)

	sorted := result.Sorted()
	require.Len(t, sorted, 3)

	// Verify lexicographic order.
	for i := 1; i < len(sorted); i++ {
		require.Less(t, sorted[i-1], sorted[i])
	}
}
