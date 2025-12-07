package template

import (
	"context"

	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

func Functions(reg pkgplugins.Registry) map[string]any {
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
				callArgs := normalizeArgs(args)

				return fn(context.Background(), callArgs...)
			}

			continue
		}

		out[name] = fn
	}

	return out
}

func normalizeArgs(args []any) []any {
	if len(args) != 1 {
		return args
	}

	switch v := args[0].(type) {
	case []any:
		return v
	case []float64:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = val
		}

		return out
	default:
		return args
	}
}
