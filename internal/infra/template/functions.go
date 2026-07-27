package template

import (
	"context"

	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

func Functions(ctx context.Context, reg pkgplugins.Registry) map[string]any {
	if reg == nil {
		reg = internalplugins.NewRegistry()
		internalplugins.RegisterBuiltins(reg)
	}

	raw := reg.Funcs()
	out := make(map[string]any, len(raw))

	for name, fn := range raw {
		if typed, ok := fn.(pkgplugins.Func); ok && typed != nil {
			fn := typed
			out[name] = func(args ...any) (any, error) {
				return fn(ctx, args...)
			}

			continue
		}

		out[name] = fn
	}

	return out
}
