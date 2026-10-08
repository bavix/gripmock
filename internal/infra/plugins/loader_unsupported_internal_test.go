package plugins

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errNotImplemented  = errors.New("plugin: not implemented")
	errRealpath        = errors.New(`plugin.Open("x.so"): realpath failed`)
	errVersionMismatch = errors.New("plugin was built with a different version of package foo")
)

func TestIsPluginUnsupported(t *testing.T) {
	t.Parallel()

	require.True(t, isPluginUnsupported(errNotImplemented))
	require.True(t, isPluginUnsupported(fmt.Errorf("wrapped: %w", errNotImplemented)))
	require.False(t, isPluginUnsupported(errRealpath))
	require.False(t, isPluginUnsupported(errVersionMismatch))
	require.False(t, isPluginUnsupported(nil))
}
