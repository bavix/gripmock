package plugintest

import (
	"context"
	"fmt"
	"reflect"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem() //nolint:gochecknoglobals
	errorType   = reflect.TypeOf((*error)(nil)).Elem()           //nolint:gochecknoglobals
)

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
		return wrapReflect(fn)
	}
}

// Call executes a Func with context propagation and returns its result. It is
// context-aware so tests can pass deadlines/logging through the same surface the
// runtime uses.
func Call(ctx context.Context, fn Func, args ...any) (any, error) {
	if fn == nil {
		return nil, nil
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

//nolint:cyclop,err113,forcetypeassert,intrange,mnd,nlreturn,wsl_v5,funlen
func wrapReflect(fn any) Func {
	val := reflect.ValueOf(fn)
	if !val.IsValid() || val.Kind() != reflect.Func {
		return nil
	}

	typ := val.Type()
	isVariadic := typ.IsVariadic()
	fixed := typ.NumIn()

	return func(ctx context.Context, args ...any) (any, error) {
		in := make([]reflect.Value, 0, len(args)+1)
		argIdx := 0

		for i := 0; i < fixed; i++ {
			paramType := typ.In(i)

			if paramType == contextType {
				in = append(in, reflect.ValueOf(ctx))
				continue
			}

			if !isVariadic || i < fixed-1 {
				valArg, err := coerceArg(args, &argIdx, paramType, typ, i)
				if err != nil {
					return nil, err
				}

				in = append(in, valArg)
				continue
			}

			elemType := paramType.Elem()
			for argIdx < len(args) {
				valArg, err := coerceArg(args, &argIdx, elemType, typ, argIdx)
				if err != nil {
					return nil, err
				}

				in = append(in, valArg)
			}
		}

		out := val.Call(in)

		switch len(out) {
		case 0:
			return nil, nil
		case 1:
			return out[0].Interface(), nil
		case 2:
			var err error
			if !out[1].IsNil() {
				if out[1].Type().Implements(errorType) {
					err = out[1].Interface().(error)
				} else {
					err = fmt.Errorf("second return value of %s does not implement error", typ)
				}
			}
			return out[0].Interface(), err
		default:
			return nil, fmt.Errorf("unsupported result count %d for %s", len(out), typ)
		}
	}
}

//nolint:exhaustive
func isNilAssignable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

//nolint:err113,wsl_v5
func coerceArg(args []any, idx *int, paramType reflect.Type, fnType reflect.Type, pos int) (reflect.Value, error) {
	if *idx >= len(args) {
		return reflect.Value{}, fmt.Errorf("not enough arguments for %s: need %d have %d", fnType, fnType.NumIn(), len(args))
	}

	raw := args[*idx]
	*idx++

	if raw == nil {
		if !isNilAssignable(paramType) {
			return reflect.Value{}, fmt.Errorf("argument %d to %s is nil but %s is not nilable", pos, fnType, paramType)
		}

		return reflect.Zero(paramType), nil
	}

	valArg := reflect.ValueOf(raw)
	if !valArg.Type().AssignableTo(paramType) {
		return reflect.Value{}, fmt.Errorf("argument %d to %s: have %s want %s", pos, fnType, valArg.Type(), paramType)
	}

	return valArg, nil
}
