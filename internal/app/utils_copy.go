package app

import (
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func deepCopyMapAny(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = deepCopyAny(v)
	}

	return dst
}

// copyForTemplates deep-copies only when the value actually carries templates:
// rendering mutates the copy in place, and a template-free payload can be
// served as-is because every consumer treats stub data as immutable.
func copyForTemplates(src any) any {
	if !template.HasTemplatesInValue(src) {
		return src
	}

	return deepCopyAny(src)
}

func deepCopyAny(src any) any {
	switch v := src.(type) {
	case map[string]any:
		return deepCopyMapAny(v)
	case []any:
		return deepCopySliceAny(v)
	default:
		return v
	}
}

func deepCopySliceAny(src []any) []any {
	if src == nil {
		return nil
	}

	dst := make([]any, len(src))
	for i, v := range src {
		switch vv := v.(type) {
		case map[string]any:
			dst[i] = deepCopyMapAny(vv)
		case []any:
			dst[i] = deepCopySliceAny(vv)
		default:
			dst[i] = v
		}
	}

	return dst
}

func deepCopyDetails(src []map[string]any) []map[string]any {
	if src == nil {
		return nil
	}

	dst := make([]map[string]any, len(src))
	for i, item := range src {
		dst[i] = deepCopyMapAny(item)
	}

	return dst
}
