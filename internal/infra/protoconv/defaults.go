// Package protoconv provides utilities for proto message conversion.
package protoconv

import (
	"maps"
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

	switch v := value.(type) {
	case string:
		return v == "" || hasSuffixIgnoreCase(v, "_UNSPECIFIED")
	case int:
		return v == 0
	case int8:
		return v == 0
	case int16:
		return v == 0
	case int32:
		return v == 0
	case int64:
		return v == 0
	case uint:
		return v == 0
	case uint8:
		return v == 0
	case uint16:
		return v == 0
	case uint32:
		return v == 0
	case uint64:
		return v == 0
	case float32:
		return v == 0.0
	case float64:
		return v == 0.0
	case bool:
		return !v // false is default
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
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
