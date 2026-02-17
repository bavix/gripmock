package descriptors_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
)

func TestRegistry_PathsSorted(t *testing.T) {
	t.Parallel()

	reg := descriptors.NewRegistry()
	pathA := filepath.Join("..", "..", "..", "examples", "projects", "calculator", "service.proto")
	pathB := filepath.Join("..", "..", "..", "examples", "projects", "greeter", "service.proto")

	fdA := mustFileDesc(t, pathA)
	fdB := mustFileDesc(t, pathB)

	reg.Register(fdB)
	reg.Register(fdA)

	paths := reg.Paths()
	require.True(t, sort.StringsAreSorted(paths))
}

func TestRegistry_ServiceIDsSorted(t *testing.T) {
	t.Parallel()

	reg := descriptors.NewRegistry()
	pathA := filepath.Join("..", "..", "..", "examples", "projects", "calculator", "service.proto")
	pathB := filepath.Join("..", "..", "..", "examples", "projects", "greeter", "service.proto")

	fdA := mustFileDesc(t, pathA)
	fdB := mustFileDesc(t, pathB)

	reg.Register(fdB)
	reg.Register(fdA)

	ids := reg.ServiceIDs()
	require.True(t, sort.StringsAreSorted(ids))
}
