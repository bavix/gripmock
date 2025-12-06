package deeply_test

import (
	"cmp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

func ranker(expect any, actual []any) []any {
	slices.SortFunc(actual, func(x, y any) int {
		return cmp.Compare(deeply.RankMatch(expect, y), deeply.RankMatch(expect, x))
	})

	return actual
}

func TestRankMatch_Simple(t *testing.T) {
	t.Parallel()

	require.Equal(t, []any{"a", "ab", "abc"}, ranker("a", []any{"a", "ab", "abc"}))
	require.Equal(t, []any{"a", "ab", "abc"}, ranker("a", []any{"a", "abc", "ab"}))

	require.Equal(t, []any{"hello", "world", "zzzzz"}, ranker("hella", []any{"world", "hello", "zzzzz"}))
	require.Equal(t, []any{"hello", "world", "zzzzz"}, ranker("hella", []any{"world", "zzzzz", "hello"}))
	require.Equal(t, []any{"hello", "world", "zzzzz"}, ranker("hella", []any{"hello", "zzzzz", "world"}))

	require.Equal(t,
		[]any{[]int{1, 2, 3}, []int{3, 2, 1}, []int{1}},
		ranker([]int{1, 2, 3}, []any{[]int{1, 2, 3}, []int{3, 2, 1}, []int{1}}))
}

func TestRankMatch_Map_Left(t *testing.T) {
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

	c := map[string]any{
		"c": map[string]any{
			"d": "d",
			"e": []int{1, 2, 3},
			"f": []string{"a", "b", "c"},
		},
		"b":      "b",
		"a":      "a",
		"name":   "gripmock",
		"cities": []string{"Istanbul", "Stalingrad"},
	}

	require.Equal(t, []any{b, c}, ranker(a, []any{c, b}))
}

func TestRankMatch_Map_Right(t *testing.T) {
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

	c := map[string]any{
		"c": map[string]any{
			"d": "d",
			"e": []int{1, 2, 3},
			"f": []string{"a", "b", "c"},
		},
		"b":      "b",
		"a":      "a",
		"name":   "gripmock",
		"cities": []string{"Istanbul", "Stalingrad"},
	}

	require.Equal(t, []any{b, c}, ranker(a, []any{c, b}))
}

func TestRankMatch_Boundary(t *testing.T) {
	t.Parallel()

	require.Equal(t, []any{nil, false, true, 0, 1}, ranker(nil, []any{false, true, 0, 1, nil}))
	require.Equal(t,
		[]any{[]string{"a", "b", "c"}, []string{"a", "b", "d"}, []string{"a", "c", "d"}},
		ranker(
			[]string{"[a]", "[b]", "[cd]"},
			[]any{[]string{"a", "b", "c"}, []string{"a", "b", "d"}, []string{"a", "c", "d"}}))

	require.Greater(t, deeply.RankMatch(map[string]any{
		"field1": "hello field1",
		"field3": "hello field3",
	}, map[string]any{
		"field1": "hello field1",
	}), 0.)

	require.Greater(t, deeply.RankMatch(map[string]any{}, map[string]any{}), 0.)
}

func TestRankMatch_RegularDigits(t *testing.T) {
	t.Parallel()

	require.Greater(t, deeply.RankMatch("[0-9]", 9), 0.)
	require.Greater(t, deeply.RankMatch("^100[1-2]{2}\\d{0,3}$", 10012), .1)

	require.Greater(t, deeply.RankMatch(
		map[any]any{"vint64": "^100[1-2]{2}\\d{0,3}$"},
		map[any]any{"vint64": 10012},
	), 0.)
}
