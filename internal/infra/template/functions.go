package template

import (
	"context"
	"reflect"

	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

type rawFuncProvider interface {
	RawFuncs() map[string]any
}

func Functions(ctx context.Context, reg pkgplugins.Registry) map[string]any {
	if reg == nil {
		reg = internalplugins.Default()
	}

	raw := reg.Funcs()

	var direct map[string]any
	if provider, ok := reg.(rawFuncProvider); ok {
		direct = provider.RawFuncs()
	}

	out := make(map[string]any, len(raw))

	for name, fn := range raw {
		if bare, ok := direct[name]; ok && callableByTemplate(bare) {
			out[name] = bare

			continue
		}

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

func callableByTemplate(fn any) bool {
	typ := reflect.TypeOf(fn)
	if typ == nil || typ.Kind() != reflect.Func {
		return false
	}

	return templateArgs(typ) && templateResults(typ)
}

func templateArgs(typ reflect.Type) bool {
	for i := range typ.NumIn() {
		in := typ.In(i)
		if typ.IsVariadic() && i == typ.NumIn()-1 {
			in = in.Elem()
		}

		if in.Kind() != reflect.Interface || in.NumMethod() != 0 {
			return false
		}
	}

	return true
}

func templateResults(typ reflect.Type) bool {
	const withError = 2

	switch typ.NumOut() {
	case 1:
		return true
	case withError:
		return typ.Out(1) == reflect.TypeFor[error]()
	default:
		return false
	}
}
