package app

import "maps"

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

func deepCopyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}

	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)

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
