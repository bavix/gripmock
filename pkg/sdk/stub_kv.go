package sdk

import (
	"github.com/cockroachdb/errors"
)

func parseKVPairsErr(kv []any, errPrefix string) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, errors.Wrapf(ErrInvalidInput, "%s: need pairs (key, value), got %d args", errPrefix, len(kv))
	}

	m := make(map[string]any, len(kv)/2) //nolint:mnd
	for i := range len(kv) / 2 {
		k, ok := kv[i*2].(string)
		if !ok {
			return nil, errors.Wrapf(ErrInvalidInput, "%s: key at %d must be string, got %T", errPrefix, i*2, kv[i*2]) //nolint:mnd
		}

		m[k] = kv[i*2+1]
	}

	return m, nil
}

func parseKVPairs(kv []any, errPrefix string) map[string]any {
	m, err := parseKVPairsErr(kv, errPrefix)
	if err != nil {
		panic(err.Error())
	}

	return m
}
