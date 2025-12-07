package runtime

import (
	"context"

	"github.com/rs/zerolog"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

//nolint:gochecknoglobals // hook storage is process-wide by design
var runtimeHooks []pkgplugins.Func

// SetHookRegistry configures runtime hooks from plugin registry.
// Hooks are selected by Group "runtime-hook".
func SetHookRegistry(reg pkgplugins.Registry) {
	if reg == nil {
		runtimeHooks = nil
		return
	}

	runtimeHooks = reg.Hooks("runtime-hook")
}

func runRuntimeHooks(ctx context.Context, args ...any) {
	if len(runtimeHooks) == 0 {
		return
	}

	for _, h := range runtimeHooks {
		if _, err := h(ctx, args...); err != nil {
			logger := zerolog.Ctx(ctx)
			if logger != nil {
				logger.Warn().Err(err).Msg("runtime hook failed")
			}
		}
	}
}

