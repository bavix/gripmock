package plugins

import (
	"context"

	"github.com/bavix/gripmock/v3/internal/infra/funcwrap"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

// wrapFunc normalizes arbitrary function shapes into the canonical Func used by the runtime registry.
func wrapFunc(fn any) pkgplugins.Func {
	return pkgplugins.WrapFunc(fn, func(fn any) pkgplugins.Func { return funcwrap.WrapReflect(fn) })
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
