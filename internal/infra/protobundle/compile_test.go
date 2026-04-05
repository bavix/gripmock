package protobundle_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/protobundle"
)

func TestCompileSingleFile(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_mixed")

	fds, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"simple.proto"},
	})
	require.NoError(t, err)
	require.NotNil(t, fds)
	require.NotEmpty(t, fds.GetFile())

	// Should contain at least our file.
	found := false

	for _, f := range fds.GetFile() {
		if f.GetName() == "simple.proto" {
			found = true

			break
		}
	}

	require.True(t, found, "simple.proto not found in descriptor set")
}

func TestCompileMultipleFiles(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root1")

	fds, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"pkg/hello.proto", "pkg/world.proto"},
	})
	require.NoError(t, err)
	require.NotNil(t, fds)

	names := make(map[string]struct{})
	for _, f := range fds.GetFile() {
		names[f.GetName()] = struct{}{}
	}

	require.Contains(t, names, "pkg/hello.proto")
	require.Contains(t, names, "pkg/world.proto")
}

func TestCompileNoFiles(t *testing.T) {
	t.Parallel()

	_, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{"/tmp"},
		Files: []string{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no files to compile")
}

func TestCompileDeterministicOrder(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root1")

	fds1, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"pkg/hello.proto", "pkg/world.proto"},
	})
	require.NoError(t, err)

	fds2, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"pkg/world.proto", "pkg/hello.proto"},
	})
	require.NoError(t, err)

	// Same descriptors regardless of input order.
	require.Len(t, fds2.GetFile(), len(fds1.GetFile()))

	for i := range fds1.GetFile() {
		require.Equal(t, fds1.GetFile()[i].GetName(), fds2.GetFile()[i].GetName())
	}
}

func TestWritePbsCompressed(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_mixed")

	fds, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"simple.proto"},
	})
	require.NoError(t, err)

	dir := t.TempDir()
	pbPath := filepath.Join(dir, "out.pb")
	pbsPath := filepath.Join(dir, "out.pbs")

	// Write both raw and compressed.
	require.NoError(t, protobundle.Write(fds, pbPath))
	require.NoError(t, protobundle.Write(fds, pbsPath))

	rawData, err := os.ReadFile(pbPath) //nolint:gosec
	require.NoError(t, err)

	compressedData, err := os.ReadFile(pbsPath) //nolint:gosec
	require.NoError(t, err)

	// Compressed output must differ from raw (S2 block header differs from protobuf wire format).
	require.NotEqual(t, rawData, compressedData)

	// Compressed should be smaller or equal (for tiny payloads S2 may add overhead, but it should still differ).
	require.NotEmpty(t, compressedData)
}

func TestWritePbsRoundTrip(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root1")

	fds, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"pkg/hello.proto", "pkg/world.proto"},
	})
	require.NoError(t, err)

	pbsPath := filepath.Join(t.TempDir(), "bundle.pbs")

	require.NoError(t, protobundle.Write(fds, pbsPath))

	compressedData, err := os.ReadFile(pbsPath) //nolint:gosec
	require.NoError(t, err)

	// Decode the compressed data back.
	decoded, err := protobundle.Decode(compressedData)
	require.NoError(t, err)
	require.NotNil(t, decoded)

	// Same number of file descriptors.
	require.Len(t, decoded.GetFile(), len(fds.GetFile()))

	// Same file names in same order.
	for i, f := range fds.GetFile() {
		require.Equal(t, f.GetName(), decoded.GetFile()[i].GetName())
	}
}

func TestDecodeInvalidData(t *testing.T) {
	t.Parallel()

	// Random garbage should fail decompression.
	_, err := protobundle.Decode([]byte("not-s2-compressed-data"))
	require.Error(t, err)
}

func TestWriteCreatesFile(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_mixed")

	fds, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"simple.proto"},
	})
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "out.pb")

	err = protobundle.Write(fds, outPath)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath) //nolint:gosec
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestWriteCreatesParentDir(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_mixed")

	fds, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"simple.proto"},
	})
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "sub", "dir", "out.pb")

	err = protobundle.Write(fds, outPath)
	require.NoError(t, err)

	_, err = os.Stat(outPath)
	require.NoError(t, err)
}

func TestWriteNoTmpFileLeft(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixturesDir(t), "root_mixed")

	fds, err := protobundle.Compile(context.Background(), protobundle.CompileParams{
		Roots: []string{root},
		Files: []string{"simple.proto"},
	})
	require.NoError(t, err)

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.pb")

	err = protobundle.Write(fds, outPath)
	require.NoError(t, err)

	// Verify no .tmp file remains.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, e := range entries {
		require.NotContains(t, e.Name(), ".tmp")
	}
}
