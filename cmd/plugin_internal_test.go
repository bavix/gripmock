package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginExtFollowsTarget(t *testing.T) {
	t.Setenv("GOOS", "windows")
	require.Equal(t, ".dll", pluginExt())

	t.Setenv("GOOS", "linux")
	require.Equal(t, ".so", pluginExt())
}

func TestWriteOverlayMapsShimIntoPackage(t *testing.T) {
	t.Parallel()

	src := filepath.Join("some", "pkg")

	overlay, cleanup, err := writeOverlay(src)
	require.NoError(t, err)

	defer cleanup()

	raw, err := os.ReadFile(overlay) //nolint:gosec // path produced by writeOverlay above
	require.NoError(t, err)

	var mapping map[string]map[string]string

	require.NoError(t, json.Unmarshal(raw, &mapping))

	shim, ok := mapping["Replace"][filepath.Join(src, shimFile)]
	require.True(t, ok)

	body, err := os.ReadFile(shim) //nolint:gosec // path taken from the overlay written above
	require.NoError(t, err)
	require.Equal(t, shimBody, string(body))

	cleanup()
	require.NoFileExists(t, overlay)
}
