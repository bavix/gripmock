package sdk

import "fmt"

func parseKVPairsErr(kv []any, errPrefix string) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("%s: need pairs (key, value), got %d args", errPrefix, len(kv))
	}

	m := make(map[string]any, len(kv)/2)
	for i := range len(kv) / 2 {
		k, ok := kv[i*2].(string)
		if !ok {
			return nil, fmt.Errorf("%s: key at %d must be string, got %T", errPrefix, i*2, kv[i*2])
		}

		m[k] = kv[i*2+1]
	}

	return m, nil
}

func parseHeaderPairsErr(kv []string, errPrefix string) (map[string]string, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("%s: need pairs (key, value), got %d args", errPrefix, len(kv))
	}

	headers := make(map[string]string, len(kv)/2)
	for i := range len(kv) / 2 {
		headers[kv[i*2]] = kv[i*2+1]
	}

	return headers, nil
}
