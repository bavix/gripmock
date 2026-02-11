package deeply_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

func runSliceOrderIgnoreChecks(t *testing.T, fn func(expect, actual any) bool) {
	t.Helper()

	t.Run("type_mismatch", func(t *testing.T) {
		t.Parallel()

		require.False(t, fn([]string{"a", "b", "c"}, []int{1, 2, 3}), "expected mismatch on different types")
	})

	t.Run("same_order_and_permutation", func(t *testing.T) {
		t.Parallel()

		require.True(t, fn([]string{"a", "b", "c"}, []string{"b", "a", "c"}), "expected match for permutation")
		require.True(t, fn([]int{1, 2, 3}, []int{1, 2, 3}), "expected match for identical slices")
		require.True(t, fn([]int{1, 2, 3}, []int{1, 3, 2}), "expected match for permutation of ints")
		require.True(t, fn([]any{1, 2, 3}, []any{1, 2, 3}), "expected match for identical any slice")
		require.True(t, fn([]any{1, 2, 3}, []any{1, 3, 2}), "expected match for permutation of any")
	})

	t.Run("missing_elements", func(t *testing.T) {
		t.Parallel()

		require.False(t, fn([]int{1, 2, 3}, []int{1, 2}), "expected mismatch when actual shorter")
		require.False(t, fn([]any{1, 2, 3}, []any{1, 2}), "expected mismatch when actual shorter (any)")
	})
}

func TestRunSliceOrderIgnoreChecks_UsesEquals(t *testing.T) {
	t.Parallel()
	runSliceOrderIgnoreChecks(t, deeply.EqualsIgnoreArrayOrder)
}

func TestRunSliceOrderIgnoreChecks_UsesContains(t *testing.T) {
	t.Parallel()
	runSliceOrderIgnoreChecks(t, deeply.ContainsIgnoreArrayOrder)
}

func TestRunSliceOrderIgnoreChecks_UsesMatches(t *testing.T) {
	t.Parallel()
	runSliceOrderIgnoreChecks(t, deeply.MatchesIgnoreArrayOrder)
}
