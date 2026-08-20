package plugins

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"strconv"

	"github.com/goccy/go-json"
)

const maxSeq = 10_000

var (
	errOddDictPairs = errors.New("dict requires an even number of arguments")
	errSetArgs      = errors.New("set requires a map, a key and a value")
	errSeqArgs      = errors.New("seq requires a count")
	errSeqTooLarge  = fmt.Errorf("seq supports at most %d indexes", maxSeq)
	errAppendArgs   = errors.New("append requires a list")
)

type jsonList []any

type jsonMap map[string]any

func (l jsonList) Format(state fmt.State, _ rune) { writeJSON(state, l) }

func (m jsonMap) Format(state fmt.State, _ rune) { writeJSON(state, m) }

func writeJSON(state fmt.State, value any) {
	if err := json.NewEncoder(state).Encode(value); err != nil {
		_, _ = state.Write([]byte(err.Error()))
	}
}

func stringOf(value any) string {
	switch text := value.(type) {
	case string:
		return text
	case json.Number:
		return text.String()
	default:
		return fmt.Sprint(value)
	}
}

func asList(value any) jsonList {
	switch v := value.(type) {
	case jsonList:
		return v
	case []any:
		return v[:len(v):len(v)]
	default:
		return nil
	}
}

func asMap(value any) jsonMap {
	switch v := value.(type) {
	case jsonMap:
		return v
	case map[string]any:
		return v
	default:
		return nil
	}
}

func arrayFuncs() map[string]any {
	return map[string]any{
		"extract": extract,
		"list":    func(items ...any) (any, error) { return list(items...), nil },
		"append":  appendItems,
		"dict":    dict,
		"set":     set,
		"seq":     seq,
	}
}

func seq(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, errSeqArgs
	}

	n, ok := convertToInt(args[0])
	if !ok || n <= 0 {
		return jsonList{}, nil
	}

	if n > maxSeq {
		return nil, errSeqTooLarge
	}

	out := make(jsonList, 0, n)
	for i := range n {
		out = append(out, json.Number(strconv.Itoa(i)))
	}

	return out, nil
}

func list(items ...any) jsonList {
	out := make(jsonList, 0, len(items))

	return append(out, items...)
}

func appendItems(args ...any) (any, error) {
	if len(args) == 0 {
		return nil, errAppendArgs
	}

	return append(asList(args[0]), args[1:]...), nil
}

func dict(pairs ...any) (any, error) {
	const pairSize = 2

	if len(pairs)%pairSize != 0 {
		return nil, errOddDictPairs
	}

	out := make(jsonMap, len(pairs)/pairSize)

	for i := pairSize - 1; i < len(pairs); i += pairSize {
		out[stringOf(pairs[i-1])] = pairs[i]
	}

	return out, nil
}

func set(args ...any) (any, error) {
	const setArity = 3

	if len(args) != setArity {
		return nil, errSetArgs
	}

	current := asMap(args[0])

	out := make(jsonMap, len(current)+1)
	maps.Copy(out, current)
	out[stringOf(args[1])] = args[2]

	return out, nil
}

func extract(collection any, key any) any {
	k := stringOf(key)

	if values := asMap(collection); values != nil {
		return values[k]
	}

	if items := asList(collection); items != nil {
		if _, ok := convertToInt(key); ok {
			return extractFromSlice(len(items), key, func(i int) any { return items[i] })
		}

		return extractFromObjects(items, k)
	}

	switch c := collection.(type) {
	case map[string]string:
		return c[k]
	case []string:
		return extractFromSlice(len(c), key, func(i int) any { return c[i] })
	}

	return nil
}

func extractFromSlice(length int, key any, getter func(int) any) any {
	idx, ok := convertToInt(key)
	if !ok || idx < 0 || idx >= length {
		return nil
	}

	return getter(idx)
}

func extractFromObjects(items jsonList, key string) any {
	out := make(jsonList, 0, len(items))

	for _, item := range items {
		if values := asMap(item); values != nil {
			if v, ok := values[key]; ok {
				out = append(out, v)
			}

			continue
		}

		if m, ok := item.(map[string]string); ok {
			if v, ok := m[key]; ok {
				out = append(out, v)
			}
		}
	}

	return out
}

func compareFuncs() map[string]any {
	cmpFn := func(a, b any) (int, bool) {
		va, okA := convertToFloat64(a)
		if !okA {
			return 0, false
		}

		vb, okB := convertToFloat64(b)
		if !okB {
			return 0, false
		}

		return cmp.Compare(va, vb), true
	}

	return map[string]any{
		"gt": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r > 0
			}

			return false
		},
		"lt": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r < 0
			}

			return false
		},
		"gte": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r >= 0
			}

			return false
		},
		"lte": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r <= 0
			}

			return false
		},
		"eq": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r == 0
			}

			return false
		},
	}
}
