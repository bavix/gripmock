// Package protoconv provides utilities for proto message conversion.
package protoconv

import (
	"reflect"
	"strings"
)

func IsDefaultValue(value any) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == "" || strings.HasSuffix(v, "_UNSPECIFIED")
	case bool:
		return !v
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() == 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() == 0
	case float32, float64:
		return reflect.ValueOf(v).Float() == 0
	case []any, map[string]any:
		return reflect.ValueOf(v).Len() == 0
	}

	return false
}
