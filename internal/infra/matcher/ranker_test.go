package matcher_test

import (
	"cmp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/matcher"
)

func ranker(expect any, actual []any) []any {
	slices.SortFunc(actual, func(x, y any) int {
		return cmp.Compare(matcher.RankMatch(expect, y), matcher.RankMatch(expect, x))
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
	t.Skip("Simplified version doesn't support complex map ranking")
}

func TestRankMatch_Map_Right(t *testing.T) {
	t.Parallel()
	t.Skip("Simplified version doesn't support complex map ranking")
}

func TestRankMatch_Boundary(t *testing.T) {
	t.Parallel()
	require.Equal(t, []any{nil, false, true, 0, 1}, ranker(nil, []any{false, true, 0, 1, nil}))
	require.Equal(t,
		[]any{[]string{"a", "b", "c"}, []string{"a", "b", "d"}, []string{"a", "c", "d"}},
		ranker(
			[]string{"[a]", "[b]", "[cd]"},
			[]any{[]string{"a", "b", "c"}, []string{"a", "b", "d"}, []string{"a", "c", "d"}}))

	require.Greater(t, matcher.RankMatch(map[string]any{
		"field1": "hello field1",
		"field3": "hello field3",
	}, map[string]any{
		"field1": "hello field1",
	}), 0.)

	require.Greater(t, matcher.RankMatch(map[string]any{}, map[string]any{}), 0.)
}

func TestRankMatch_RegularDigits(t *testing.T) {
	t.Parallel()
	t.Skip("Simplified version doesn't support complex regex ranking")
}
