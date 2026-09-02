package stuber

import "strings"

func normalizeHeaderKeys(headers map[string]any) map[string]any {
	if !hasUpperKey(headers) {
		return headers
	}

	normalized := make(map[string]any, len(headers))
	for key, value := range headers {
		normalized[strings.ToLower(key)] = value
	}

	return normalized
}

func normalizeStubHeaders(headers *InputHeader) {
	headers.Equals = normalizeHeaderKeys(headers.Equals)
	headers.Contains = normalizeHeaderKeys(headers.Contains)
	headers.Matches = normalizeHeaderKeys(headers.Matches)
	headers.Glob = normalizeHeaderKeys(headers.Glob)

	for i := range headers.AnyOf {
		alt := &headers.AnyOf[i]
		alt.Equals = normalizeHeaderKeys(alt.Equals)
		alt.Contains = normalizeHeaderKeys(alt.Contains)
		alt.Matches = normalizeHeaderKeys(alt.Matches)
		alt.Glob = normalizeHeaderKeys(alt.Glob)
	}
}

func hasUpperKey(headers map[string]any) bool {
	for key := range headers {
		for i := range len(key) {
			if key[i] >= 'A' && key[i] <= 'Z' {
				return true
			}
		}
	}

	return false
}
