//go:build !windows

package plugins

import (
	"context"
	"errors"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

var errDLLUnsupported = errors.New("dll plugins are supported on windows only; use a .so plugin")

func loadDLL(_ context.Context, _ pkgplugins.Registry, _ string) error {
	return errDLLUnsupported
}
