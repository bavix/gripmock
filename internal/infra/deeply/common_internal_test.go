package deeply

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/url"
	"reflect"
	"testing"

	"github.com/spf13/cast"
	"github.com/stretchr/testify/require"
)

// TestStructuralValuesNeedNoDeepEqual pins the assumption that lets the public
// matchers skip reflect.DeepEqual for maps and slices: whenever DeepEqual would
// have said yes, the dedicated map or slice path already did.
var errStringify = errors.New("boom")

func TestStructuralValuesNeedNoDeepEqual(t *testing.T) {
	t.Parallel()

	values := []any{
		map[string]any{},
		map[string]any{"name": "user-1"},
		map[string]any{"name": "user-1", "age": 42.0},
		map[string]any{"nested": map[string]any{"a": "b"}},
		map[string]any{"list": []any{"a", "b"}},
		map[string]string{"name": "user-1"},
		[]any{},
		[]any{"a", "b"},
		[]any{map[string]any{"a": "b"}},
		[]string{"a", "b"},
		[]byte("payload"),
		map[string]any(nil),
		[]any(nil),
	}

	for _, expect := range values {
		require.True(t, structural(expect))

		for _, actual := range values {
			if !reflect.DeepEqual(expect, actual) {
				continue
			}

			require.True(t, EqualsIgnoreArrayOrder(expect, actual))
			require.True(t, ContainsIgnoreArrayOrder(expect, actual))
			require.True(t, MatchesIgnoreArrayOrder(expect, actual))
		}
	}
}

func TestStructuralRejectsLeaves(t *testing.T) {
	t.Parallel()

	for _, value := range []any{nil, "text", 1, 1.5, true, errStringify, struct{ A int }{1}, [2]int{1, 2}} {
		require.False(t, structural(value))
	}
}

// TestStringifyMatchesCast keeps the allocation-free conversion in step with
// the cast package it replaced.
func TestStringifyMatchesCast(t *testing.T) {
	t.Parallel()

	convertible := []any{
		"text", "", nil, true, false,
		[]byte("bytes"), int(1), int8(2), int16(3), int32(4), int64(5),
		uint(6), uint8(7), uint16(8), uint32(9), uint64(10),
		float32(1.5), float64(2.25), json.Number("42"),
		template.HTML("<b>"), template.URL("/path"), template.JS("x"), template.CSS("a{}"),
		errStringify, &url.URL{Host: "example.com"},
	}

	for _, value := range convertible {
		want, err := cast.ToStringE(value)
		require.NoError(t, err)

		got, ok := stringify(value)
		require.True(t, ok)
		require.Equal(t, want, got)
	}

	for _, value := range []any{map[string]any{"a": 1}, []any{1}, struct{}{}, complex(1, 2)} {
		_, err := cast.ToStringE(value)
		require.Error(t, err)

		_, ok := stringify(value)
		require.False(t, ok)
	}
}
