package funcwrap

import (
	"context"
	"fmt"
	"reflect"
)

var ContextType = reflect.TypeFor[context.Context]() //nolint:gochecknoglobals

// WrapReflect uses reflection to convert any function into the canonical
// func(context.Context, ...any) (any, error) shape. It injects context.Context
// parameters and coerces arguments by type. Returns nil for non-functions.
func WrapReflect(fn any) func(context.Context, ...any) (any, error) {
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

		for i := range fixed {
			paramType := typ.In(i)

			if paramType == ContextType {
				in = append(in, reflect.ValueOf(ctx))

				continue
			}

			if !isVariadic || i < fixed-1 {
				valArg, err := CoerceArg(args, &argIdx, paramType, typ, i)
				if err != nil {
					return nil, err
				}

				in = append(in, valArg)

				continue
			}

			elemType := paramType.Elem()
			for argIdx < len(args) {
				valArg, err := CoerceArg(args, &argIdx, elemType, typ, argIdx)
				if err != nil {
					return nil, err
				}

				in = append(in, valArg)
			}
		}

		out := val.Call(in)

		return handleReflectCallResult(out, typ)
	}
}

//nolint:err113,nilnil
func handleReflectCallResult(out []reflect.Value, typ reflect.Type) (any, error) {
	switch len(out) {
	case 0:
		return nil, nil
	case 1:
		return out[0].Interface(), nil
	case 2: //nolint:mnd
		var err error

		if !out[1].IsNil() {
			if errVal, ok := out[1].Interface().(error); ok {
				err = errVal
			} else {
				err = fmt.Errorf("second return value of %s does not implement error", typ)
			}
		}

		return out[0].Interface(), err
	default:
		return nil, fmt.Errorf("unsupported result count %d for %s", len(out), typ)
	}
}

// IsNilAssignable returns true if the type can hold a nil value.
//
//nolint:exhaustive
func IsNilAssignable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

// CoerceArg coerces a raw argument to the expected parameter type.
//
//nolint:err113
func CoerceArg(args []any, idx *int, paramType reflect.Type, fnType reflect.Type, pos int) (reflect.Value, error) {
	if *idx >= len(args) {
		return reflect.Value{}, fmt.Errorf("not enough arguments for %s: need %d have %d", fnType, fnType.NumIn(), len(args))
	}

	raw := args[*idx]
	*idx++

	if raw == nil {
		if !IsNilAssignable(paramType) {
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
