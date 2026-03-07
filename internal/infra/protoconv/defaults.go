// Package protoconv provides utilities for proto message conversion.
package protoconv

import (
	"maps"
	"reflect"
	"strings"
)

// DefaultValueHandling specifies how to handle default values during conversion.
type DefaultValueHandling int

const (
	// IncludeDefaults includes all fields including default values (proto3 behavior).
	// This is the default behavior for backward compatibility.
	IncludeDefaults DefaultValueHandling = iota

	// ExcludeDefaults excludes fields with default values.
	// Use this when you want to show only explicitly set fields.
	ExcludeDefaults
)

// IsDefaultValue checks if a value is a default/empty value for proto3.
// Returns true for: nil, empty string, 0, 0.0, false, empty collections,
// and enum values ending with _UNSPECIFIED.
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

// hasSuffixIgnoreCase checks if string ends with suffix (case-sensitive for enum values).
// We use case-sensitive check because proto enum values are UPPER_CASE.
func hasSuffixIgnoreCase(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// FilterDefaultValues removes fields with default/empty values from a map.
// Returns a new map without modifying the original.
func FilterDefaultValues(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	filtered := make(map[string]any, len(input))
	for k, v := range input {
		if !IsDefaultValue(v) {
			filtered[k] = v
		}
	}

	return filtered
}

// ConvertMap converts a map according to the specified default value handling.
// If handling is IncludeDefaults, returns the original map (or a copy if needed).
// If handling is ExcludeDefaults, returns a new map with default values filtered out.
func ConvertMap(input map[string]any, handling DefaultValueHandling) map[string]any {
	if input == nil {
		return nil
	}

	switch handling {
	case IncludeDefaults:
		// Return a shallow copy to avoid accidental modification
		return maps.Clone(input)
	case ExcludeDefaults:
		return FilterDefaultValues(input)
	default:
		// Default behavior: include defaults for backward compatibility
		return maps.Clone(input)
	}
}
