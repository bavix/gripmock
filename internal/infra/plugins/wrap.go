package plugins

import (
	"context"

	"github.com/bavix/gripmock/v3/internal/infra/funcwrap"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

// wrapFunc normalizes arbitrary function shapes into the canonical Func used by the runtime registry.
func wrapFunc(fn any) pkgplugins.Func {
	switch f := fn.(type) {
	case pkgplugins.Func:
		return f
	case func(context.Context, ...any) (any, error):
		return f
	case func(...any) any:
		return func(_ context.Context, args ...any) (any, error) {
			return f(args...), nil
		}
	case func(...any) (any, error):
		return func(_ context.Context, args ...any) (any, error) {
			return f(args...)
		}
	default:
		return funcwrap.WrapReflect(fn)
	}
}

func wrapDecorator(fn any) func(pkgplugins.Func) pkgplugins.Func {
	switch f := fn.(type) {
	case func(pkgplugins.Func) pkgplugins.Func:
		return f
	case func(func(context.Context, ...any) (any, error)) func(context.Context, ...any) (any, error):
		return func(base pkgplugins.Func) pkgplugins.Func {
			return f(base)
		}
	default:
		return nil
	}
}
