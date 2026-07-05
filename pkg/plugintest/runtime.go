package plugintest

import (
	"context"

	"github.com/cockroachdb/errors"

	"github.com/bavix/gripmock/v3/internal/infra/funcwrap"
)

// ErrNilFunc is returned when Call receives a nil function.
var ErrNilFunc = errors.New("plugintest: nil function")

// Wrap converts a testing helper or plugin-style function into the canonical Func
// so tests can exercise callbacks without rewriting them. It accepts common shapes
// used in plugins and falls back to reflection with context injection, rejecting
// mismatched arity or types with clear errors.
func Wrap(fn any) Func {
	switch f := fn.(type) {
	case Func:
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

// Call executes a Func with context propagation and returns its result. It is
// context-aware so tests can pass deadlines/logging through the same surface the
// runtime uses.
func Call(ctx context.Context, fn Func, args ...any) (any, error) {
	if fn == nil {
		return nil, ErrNilFunc
	}

	return fn(ctx, args...)
}

// Decorate wraps base with decorator while preserving the canonical signature.
// In tests this is useful to assert decoration order or to inject spies.
func Decorate(base Func, decorator func(Func) Func) Func {
	if base == nil || decorator == nil {
		return base
	}

	return decorator(base)
}

// LookupFunc fetches a Func from a Registry by name for assertion or invocation
// without reaching into registry internals.
func LookupFunc(reg Registry, name string) (Func, bool) {
	if reg == nil || name == "" {
		return nil, false
	}

	fn, ok := reg.Funcs()[name]
	if !ok {
		return nil, false
	}

	casted, ok := fn.(Func)

	return casted, ok
}
