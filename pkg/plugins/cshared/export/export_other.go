//go:build !windows || !cgo

// Package export carries the C entry points a plugin DLL must expose.
// Without cgo, or outside windows, there is nothing to export.
package export

import (
	"github.com/bavix/gripmock/v3/pkg/plugins"
	"github.com/bavix/gripmock/v3/pkg/plugins/cshared"
)

// Use registers the plugin the exported entry points answer for.
func Use(register func(plugins.Registry)) {
	cshared.Use(register)
}
