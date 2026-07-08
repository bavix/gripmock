package protoconv

import (
	"reflect"
	"strings"
)

func IsDefaultValue(value any) bool {
	if value == nil {
		return true
	}

	if v, ok := value.(string); ok {
		return v == "" || hasSuffixIgnoreCase(v, "_UNSPECIFIED")
	}

	return isZeroReflectValue(reflect.ValueOf(value))
}

func isZeroReflectValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return value.Len() == 0
	case reflect.Invalid,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Pointer,
		reflect.String,
		reflect.Struct,
		reflect.UnsafePointer:
		return false
	default:
		return false
	}
}

func hasSuffixIgnoreCase(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}
