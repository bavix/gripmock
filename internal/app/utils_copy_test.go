package app //nolint:testpackage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestDeepCopyMapAny(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		require.Nil(t, deepCopyMapAny(nil))
	})

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()

		src := map[string]any{}
		dst := deepCopyMapAny(src)
		require.NotNil(t, dst)
		require.Empty(t, dst)
	})

	t.Run("flat map", func(t *testing.T) {
		t.Parallel()

		src := map[string]any{"a": 1, "b": "two", "c": true}
		dst := deepCopyMapAny(src)
		require.Equal(t, src, dst)
		src["a"] = 999

		require.Equal(t, 1, dst["a"])
	})

	t.Run("nested map", func(t *testing.T) {
		t.Parallel()

		src := map[string]any{
			"outer": map[string]any{"inner": 42},
		}
		dst := deepCopyMapAny(src)
		require.Equal(t, src, dst)
		outer, ok := src["outer"].(map[string]any)
		require.True(t, ok)

		outer["inner"] = 999

		dstOuter, ok := dst["outer"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, 42, dstOuter["inner"])
	})

	t.Run("map with slice", func(t *testing.T) {
		t.Parallel()

		src := map[string]any{"arr": []any{1, 2, 3}}
		dst := deepCopyMapAny(src)
		require.Equal(t, src, dst)
		arr, ok := src["arr"].([]any)
		require.True(t, ok)

		arr[0] = 999

		dstArr, ok := dst["arr"].([]any)
		require.True(t, ok)
		require.Equal(t, 1, dstArr[0])
	})

	t.Run("deeply nested", func(t *testing.T) {
		t.Parallel()

		src := map[string]any{
			"a": map[string]any{
				"b": []any{map[string]any{"c": 1}},
			},
		}
		dst := deepCopyMapAny(src)
		require.Equal(t, src, dst)
		a, ok := src["a"].(map[string]any)
		require.True(t, ok)
		b, ok := a["b"].([]any)
		require.True(t, ok)
		inner, ok := b[0].(map[string]any)
		require.True(t, ok)

		inner["c"] = 999

		dstA, ok := dst["a"].(map[string]any)
		require.True(t, ok)
		dstB, ok := dstA["b"].([]any)
		require.True(t, ok)
		dstInner, ok := dstB[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, 1, dstInner["c"])
	})
}

func TestDeepCopySliceAny(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		require.Nil(t, deepCopySliceAny(nil))
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		src := []any{}
		dst := deepCopySliceAny(src)
		require.NotNil(t, dst)
		require.Empty(t, dst)
	})

	t.Run("flat slice", func(t *testing.T) {
		t.Parallel()

		src := []any{1, "two", true}
		dst := deepCopySliceAny(src)
		require.Equal(t, src, dst)
		src[0] = 999

		require.Equal(t, 1, dst[0])
	})

	t.Run("slice with map", func(t *testing.T) {
		t.Parallel()

		src := []any{map[string]any{"k": "v"}}
		dst := deepCopySliceAny(src)
		require.Equal(t, src, dst)
		m, ok := src[0].(map[string]any)
		require.True(t, ok)

		m["k"] = "modified"

		dstM, ok := dst[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "v", dstM["k"])
	})

	t.Run("slice with nested slice", func(t *testing.T) {
		t.Parallel()

		src := []any{[]any{1, 2}, []any{3, 4}}
		dst := deepCopySliceAny(src)
		require.Equal(t, src, dst)
		s0, ok := src[0].([]any)
		require.True(t, ok)

		s0[0] = 999

		d0, ok := dst[0].([]any)
		require.True(t, ok)
		require.Equal(t, 1, d0[0])
	})
}

func TestDeepCopyStringMap(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		require.Nil(t, deepCopyStringMap(nil))
	})

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()

		src := map[string]string{}
		dst := deepCopyStringMap(src)
		require.NotNil(t, dst)
		require.Empty(t, dst)
	})

	t.Run("map with values", func(t *testing.T) {
		t.Parallel()

		src := map[string]string{"k1": "v1", "k2": "v2"}
		dst := deepCopyStringMap(src)
		require.Equal(t, src, dst)
		src["k1"] = "modified"

		require.Equal(t, "v1", dst["k1"])
	})
}
