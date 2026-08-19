package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func newLoader(t *testing.T) (*Extender, *stuber.Budgerigar) {
	t.Helper()

	budgerigar := stuber.NewBudgerigar()

	return NewStub(budgerigar, yaml2json.New(plugintest.NewRegistry()), nil), budgerigar
}

func writeStubFile(t *testing.T, dir, name, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

const jsonStub = `{
	"service": "svc.Service",
	"method": "Method",
	"input": {"equals": {"id": "1"}},
	"output": {"data": {"ok": true}}
}`

const yamlStub = `- service: svc.Service
  method: Other
  input:
    equals:
      id: "2"
  output:
    data:
      ok: true
`

func TestLoaderReadsJSONYAMLAndNestedDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeStubFile(t, dir, "one.json", jsonStub)
	writeStubFile(t, dir, "two.yaml", yamlStub)
	writeStubFile(t, dir, "nested/three.yml", yamlStub)
	writeStubFile(t, dir, "notes.txt", "ignored")
	writeStubFile(t, dir, "notes.md", jsonStub)

	loader, budgerigar := newLoader(t)
	loader.readFromPath(t.Context(), dir)

	require.Len(t, budgerigar.All(), 3, "only .json/.yaml/.yml files are stubs")
}

func TestLoaderSkipsUnreadableFileAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeStubFile(t, dir, "good.json", jsonStub)
	writeStubFile(t, dir, "broken.json", "{not json")

	loader, budgerigar := newLoader(t)
	loader.readFromPath(t.Context(), dir)

	require.Len(t, budgerigar.All(), 1, "one bad file must not take the directory down")
}

func TestLoaderAssignsIDsAndKeepsThemStableOnReload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeStubFile(t, dir, "one.json", jsonStub)

	loader, budgerigar := newLoader(t)
	loader.readFromPath(t.Context(), dir)

	first := budgerigar.All()
	require.Len(t, first, 1)
	require.NotEqual(t, uuid.Nil, first[0].ID)

	loader.readByFile(t.Context(), path)

	second := budgerigar.All()
	require.Len(t, second, 1, "reloading the same file must not duplicate its stubs")
	require.Equal(t, first[0].ID, second[0].ID, "a stub keeps its generated ID across reloads")
}

func TestLoaderReplacesStubsWhenAFileShrinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeStubFile(t, dir, "many.json", `[
		{"service": "svc.Service", "method": "A", "input": {"equals": {"id": "1"}}, "output": {"data": {"ok": true}}},
		{"service": "svc.Service", "method": "B", "input": {"equals": {"id": "2"}}, "output": {"data": {"ok": true}}}
	]`)

	loader, budgerigar := newLoader(t)
	loader.readFromPath(t.Context(), dir)
	require.Len(t, budgerigar.All(), 2)

	require.NoError(t, os.WriteFile(path, []byte(`[
		{"service": "svc.Service", "method": "A", "input": {"equals": {"id": "1"}}, "output": {"data": {"ok": true}}}
	]`), 0o600))

	loader.readByFile(t.Context(), path)

	left := budgerigar.All()
	require.Len(t, left, 1, "the stub dropped from the file must be dropped from storage")
	require.Equal(t, "A", left[0].Method)
}

func TestLoaderDropsStubsOfAFileThatBecameUnreadable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeStubFile(t, dir, "one.json", jsonStub)

	loader, budgerigar := newLoader(t)
	loader.readFromPath(t.Context(), dir)
	require.Len(t, budgerigar.All(), 1)

	require.NoError(t, os.WriteFile(path, []byte("{broken"), 0o600))
	loader.readByFile(t.Context(), path)

	require.Empty(t, budgerigar.All(), "a file that stopped parsing takes its stubs with it")
}

func TestLoaderWarnsAndCollapsesDuplicateIDs(t *testing.T) {
	t.Parallel()

	const pinned = "44444444-4444-4444-4444-444444444444"

	dir := t.TempDir()
	writeStubFile(t, dir, "first.json", `{
		"id": "`+pinned+`",
		"service": "svc.Service", "method": "A",
		"input": {"equals": {"id": "1"}}, "output": {"data": {"ok": true}}
	}`)
	writeStubFile(t, dir, "second.json", `{
		"id": "`+pinned+`",
		"service": "svc.Service", "method": "B",
		"input": {"equals": {"id": "2"}}, "output": {"data": {"ok": true}}
	}`)

	var logs bytes.Buffer

	logger := zerolog.New(&logs)
	ctx := logger.WithContext(t.Context())

	loader, budgerigar := newLoader(t)
	loader.readFromPath(ctx, dir)

	stubs := budgerigar.All()
	require.Len(t, stubs, 1, "the storage is keyed by ID, so the second stub replaces the first")
	require.Equal(t, pinned, stubs[0].ID.String())
	require.Contains(t, logs.String(), "duplicate stub ID")
}

func TestLoaderReadsASingleFilePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeStubFile(t, dir, "one.json", jsonStub)

	loader, budgerigar := newLoader(t)
	loader.readFromPath(t.Context(), path)

	require.Len(t, budgerigar.All(), 1, "the stub path may be a file, not only a directory")
}

func TestLoaderTracksAFileByOneKeyHoweverItIsSpelled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeStubFile(t, dir, "one.json", jsonStub)

	loader, budgerigar := newLoader(t)
	loader.readFromPath(t.Context(), dir)
	require.Len(t, budgerigar.All(), 1)

	loader.readByFile(t.Context(), filepath.Join(dir, ".", "one.json"))
	require.Len(t, budgerigar.All(), 1, "an equivalent path must not add a second copy")

	loader.readByFile(t.Context(), dir+string(filepath.Separator)+"."+string(filepath.Separator)+"one.json")
	require.Len(t, budgerigar.All(), 1,
		"the loader and the watcher reach the same file by different routes")
}
