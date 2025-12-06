package deeply_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

func TestMatches_Simple(t *testing.T) {
	t.Parallel()

	require.True(t, deeply.Matches("a", "a"))
	require.False(t, deeply.Matches("a", "b"))

	require.True(t, deeply.Matches([]int{1, 2, 3}, []int{1, 2, 3}))
	require.False(t, deeply.Matches([]int{1, 2, 3}, []int{1, 3, 2}))
}

func TestMatches_Map_Left(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"a": "a",
		"b": "b",
		"c": map[string]any{
			"f": []string{"a", "b", "c"},
			"d": "d",
			"e": []int{1, 2, 3},
		},
		"name":   "^grip.*$",
		"cities": []string{"Jakarta", "Istanbul", ".*grad$"},
	}

	b := map[string]any{
		"c": map[string]any{
			"d": "d",
			"e": []int{1, 2, 3},
			"f": []string{"a", "b", "c"},
		},
		"b":      "b",
		"a":      "a",
		"name":   "gripmock",
		"cities": []string{"Jakarta", "Istanbul", "Stalingrad"},
	}

	require.True(t, deeply.Matches(a, b))

	delete(a, "a")

	require.True(t, deeply.Matches(a, b))

	a["a"] = true

	require.False(t, deeply.Matches(a, b))
}

func TestMatches_Map_Right(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"a": "[a-z]",
		"b": "b",
		"c": map[string]any{
			"f": []string{"[a-z]", "[0-9]", "c"},
			"d": "d",
			"e": []int{1, 2, 3},
		},
	}

	b := map[string]any{
		"c": map[string]any{
			"d": "d",
			"e": []int{1, 2, 3},
			"f": []string{"d", "1", "c"},
		},
		"b": "b",
		"a": "c",
	}

	require.True(t, deeply.Matches(a, b))

	delete(b, "a")

	require.False(t, deeply.Matches(a, b))

	b["a"] = true

	require.False(t, deeply.Matches(a, b))
}

func TestMatches_Slices_Left(t *testing.T) {
	t.Parallel()

	require.True(t, deeply.Matches([]int{1, 2, 3}, []int{1, 2, 3}))

	require.False(t, deeply.Matches([]int{1, 3, 2}, []int{1, 2, 3}))
	require.False(t, deeply.Matches([]int{1, 2}, []int{1, 2, 3}))

	require.True(t, deeply.Matches([]any{1, 2, 3}, []any{1, 2, 3}))

	require.False(t, deeply.Matches([]any{1, 3, 2}, []any{1, 2, 3}))
	require.False(t, deeply.Matches([]any{1, 2}, []any{1, 2, 3}))
}

func TestMatches_Slices_Right(t *testing.T) {
	t.Parallel()

	require.False(t, deeply.Matches([]string{"^hello$"}, []string{"hell!"}))

	require.True(t, deeply.Matches([]int{1, 2, 3}, []int{1, 2, 3}))

	require.False(t, deeply.Matches([]int{1, 2, 3}, []int{1, 3, 2}))
	require.False(t, deeply.Matches([]int{1, 2, 3}, []int{1, 2}))

	require.True(t, deeply.Matches([]any{1, 2, 3}, []any{1, 2, 3}))

	require.False(t, deeply.Matches([]any{1, 2, 3}, []any{1, 3, 2}))
	require.False(t, deeply.Matches([]any{1, 2, 3}, []any{1, 2}))
}

func TestMatches_Slices_OrderIgnore(t *testing.T) {
	t.Parallel()

	runSliceOrderIgnoreChecks(t, deeply.MatchesIgnoreArrayOrder)
}

func TestMatches_RegularDigits(t *testing.T) {
	t.Parallel()

	require.True(t, deeply.Matches("[0-9]", 9))
	require.True(t, deeply.Matches("^100[1-2]{2}\\d{0,3}$", 10012))

	require.True(t, deeply.Matches(
		map[any]any{"vint64": "^100[1-2]{2}\\d{0,3}$"},
		map[any]any{"vint64": 10012},
	))
}

//nolint:funlen
func TestMatches_Boundary_True(t *testing.T) {
	t.Parallel()

	require.True(t, deeply.Matches([]string{"[a]", "[b]", "[cd]"}, []string{"a", "b", "d"}))
	require.True(t, deeply.Matches(nil, nil))

	require.True(t, deeply.Matches(map[string]any{
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

	require.True(t, deeply.Matches(map[string]any{
		"key": "[a-z]{3}ue",
		"greetings": map[string]any{
			"hola":    1,
			"merhaba": true,
			"hello":   "^he[l]{2,}o$",
		},
		"cities": []any{
			"Istanbul",
			"Jakarta",
			".*",
		},
		"mixed": []any{
			5.5,
			false,
			".*",
		},
	}, map[string]any{
		"key": "value",
		"greetings": map[string]any{
			"hola":    1,
			"merhaba": true,
			"hello":   "helllllo",
		},
		"cities": []any{
			"Istanbul",
			"Jakarta",
			"Gotham",
		},
		"mixed": []any{
			5.5,
			false,
			"Gotham",
		},
	}))
}

func TestMatches_Boundary_False(t *testing.T) {
	t.Parallel()

	require.False(t, deeply.Matches([]string{"a", "a", "a"}, []string{"a", "b", "c"}))
	require.False(t, deeply.Matches([]string{"a", "b", "c"}, []string{"a", "a", "a"}))
	require.False(t, deeply.Matches(nil, false))

	require.False(t, deeply.Matches(map[string]any{
		"key": "[a-z]{3}ue",
		"greetings": map[string]any{
			"hola":    1,
			"merhaba": true,
			"hello":   "^he[l]{2,}o$",
		},
		"cities": []any{
			"Istanbul",
			"Jakarta",
			".*",
		},
		"mixed": []any{
			5.5,
			false,
			".*",
		},
	}, map[string]any{
		"key": "value",
		"greetings": map[string]any{
			"hola":    1,
			"merhaba": true,
			"hello":   "helllllo",
		},
		"cities": []any{
			"Istanbul",
			"Jakarta",
			"Gotham",
		},
		"mixed": []any{
			false,
			5.5,
			"Gotham",
		},
	}))
}
