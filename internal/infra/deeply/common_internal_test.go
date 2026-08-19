package deeply

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestStringifyConversions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value any
		want  string
	}{
		{"text", "text"},
		{"", ""},
		{nil, ""},
		{true, "true"},
		{false, "false"},
		{[]byte("bytes"), "bytes"},
		{int(1), "1"},
		{int8(2), "2"},
		{int16(3), "3"},
		{int32(4), "4"},
		{int64(5), "5"},
		{uint(6), "6"},
		{uint8(7), "7"},
		{uint16(8), "8"},
		{uint32(9), "9"},
		{uint64(10), "10"},
		{float32(1.5), "1.5"},
		{float64(2.25), "2.25"},
		{json.Number("42"), "42"},
		{template.HTML("<b>"), "<b>"},
		{template.URL("/path"), "/path"},
		{template.JS("x"), "x"},
		{template.CSS("a{}"), "a{}"},
		{errStringify, errStringify.Error()},
		{&url.URL{Host: "example.com"}, "//example.com"},
	}

	for _, tc := range cases {
		got, ok := stringify(tc.value)
		require.True(t, ok, "%T", tc.value)
		require.Equal(t, tc.want, got, "%T", tc.value)
	}

	for _, value := range []any{map[string]any{"a": 1}, []any{1}, struct{}{}, complex(1, 2)} {
		_, ok := stringify(value)
		require.False(t, ok, "%T", value)
	}
}
