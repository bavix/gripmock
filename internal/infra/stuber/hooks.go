package stuber

import (
	"context"

	"github.com/rs/zerolog"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

//nolint:gochecknoglobals // hooks are process-wide by design
var matcherHooks []pkgplugins.Func

// SetMatcherHooks wires matcher hooks (group "matcher-hook") from registry.
func SetMatcherHooks(reg pkgplugins.Registry) {
	if reg == nil {
		matcherHooks = nil
		return
	}

	matcherHooks = reg.Hooks("matcher-hook")
}

func runMatcherHooks(ctx context.Context, query Query, stub *Stub) {
	if len(matcherHooks) == 0 {
		return
	}

	for _, h := range matcherHooks {
		if _, err := h(ctx, query, stub); err != nil {
			logger := zerolog.Ctx(ctx)
			if logger != nil {
				logger.Warn().Err(err).Msg("matcher hook failed")
			}
		}
	}
}
