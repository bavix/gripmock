package app

func deepCopyMapAny(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		switch vv := v.(type) {
		case map[string]any:
			dst[k] = deepCopyMapAny(vv)
		case []any:
			dst[k] = deepCopySliceAny(vv)
		default:
			dst[k] = v
		}
	}

	return dst
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
	for k, v := range src {
		dst[k] = v
	}

	return dst
}
