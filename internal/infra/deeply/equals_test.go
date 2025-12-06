package deeply_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

func TestEquals_Simple(t *testing.T) {
	t.Parallel()

	require.True(t, deeply.Equals("a", "a"))
	require.False(t, deeply.Equals("a", "b"))

	require.True(t, deeply.Equals([]int{1, 2, 3}, []int{1, 2, 3}))
}

func TestEquals_Map_Left(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"a": "a",
		"b": "b",
		"c": map[string]any{
			"f": []string{"a", "b", "c"},
			"d": "d",
			"e": []int{1, 2, 3},
		},
	}

	b := map[string]any{
		"c": map[string]any{
			"d": "d",
			"e": []int{1, 2, 3},
			"f": []string{"a", "b", "c"},
		},
		"b": "b",
		"a": "a",
	}

	require.True(t, deeply.Equals(a, b))

	delete(a, "a")

	require.False(t, deeply.Equals(a, b))

	a["a"] = true

	require.False(t, deeply.Equals(a, b))
}

func TestEquals_Map_Right(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"a": "a",
		"b": "b",
		"c": map[string]any{
			"f": []string{"a", "b", "c"},
			"d": "d",
			"e": []int{1, 2, 3},
		},
	}

	b := map[string]any{
		"c": map[string]any{
			"d": "d",
			"e": []int{1, 2, 3},
			"f": []string{"a", "b", "c"},
		},
		"b": "b",
		"a": "a",
	}

	require.True(t, deeply.Equals(a, b))

	delete(b, "a")

	require.False(t, deeply.Equals(a, b))

	b["a"] = true

	require.False(t, deeply.Equals(a, b))
}

func TestEquals_Slices_Left(t *testing.T) {
	t.Parallel()

	require.True(t, deeply.Equals([]int{1, 2, 3}, []int{1, 2, 3}))

	require.False(t, deeply.Equals([]int{1, 3, 2}, []int{1, 2, 3}))
	require.False(t, deeply.Equals([]int{1, 2}, []int{1, 2, 3}))

	require.True(t, deeply.Equals([]any{1, 2, 3}, []any{1, 2, 3}))

	require.False(t, deeply.Equals([]any{1, 3, 2}, []any{1, 2, 3}))
	require.False(t, deeply.Equals([]any{1, 2}, []any{1, 2, 3}))
}

func TestEquals_Slices_Right(t *testing.T) {
	t.Parallel()

	require.True(t, deeply.Equals([]int{1, 2, 3}, []int{1, 2, 3}))

	require.False(t, deeply.Equals([]int{1, 2, 3}, []int{1, 3, 2}))
	require.False(t, deeply.Equals([]int{1, 2, 3}, []int{1, 2}))

	require.True(t, deeply.Equals([]any{1, 2, 3}, []any{1, 2, 3}))

	require.False(t, deeply.Equals([]any{1, 2, 3}, []any{1, 3, 2}))
	require.False(t, deeply.Equals([]any{1, 2, 3}, []any{1, 2}))
}

func TestEquals_Slices_OrderIgnore(t *testing.T) {
	t.Parallel()

	runSliceOrderIgnoreChecks(t, deeply.EqualsIgnoreArrayOrder)
}

func TestEquals_Boundary(t *testing.T) {
	t.Parallel()

	require.False(t, deeply.Equals([]string{"a", "a", "a"}, []string{"a", "b", "c"}))
	require.False(t, deeply.Equals([]string{"a", "b", "c"}, []string{"a", "a", "a"}))
	require.False(t, deeply.Equals(nil, false))

	require.True(t, deeply.Equals(nil, nil))

	require.True(t, deeply.Equals(map[string]any{
		"name": "Afra Gokce",
		"age":  1,
		"girl": true,
		"null": nil,
		"greetings": map[string]any{
			"hola":    "mundo",
			"merhaba": "dunya",
		},
		"cities": []any{
			"Istanbul",
			"Jakarta",
		},
	}, map[string]any{
		"name": "Afra Gokce",
		"age":  1,
		"girl": true,
		"null": nil,
		"greetings": map[string]any{
			"hola":    "mundo",
			"merhaba": "dunya",
		},
		"cities": []any{
			"Istanbul",
			"Jakarta",
		},
	}))
}
