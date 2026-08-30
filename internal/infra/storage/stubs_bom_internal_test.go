package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

func TestReadStubAcceptsFilesWrittenWithABOM(t *testing.T) {
	t.Parallel()

	body := "\xef\xbb\xbf- service: a.B\n  method: C\n  input:\n    contains:\n      id: 1\n  output:\n    data:\n      ok: true\n"

	path := filepath.Join(t.TempDir(), "bom.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	extender := NewStub(stuber.NewBudgerigar(), yaml2json.New(internalplugins.Default()), nil, nil)

	stubs, err := extender.readStub(t.Context(), path)
	require.NoError(t, err)
	require.Len(t, stubs, 1)
	require.Equal(t, "a.B", stubs[0].Service)
}
