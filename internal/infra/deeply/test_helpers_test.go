package deeply_test

import (
	"testing"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

func runSliceOrderIgnoreChecks(t *testing.T, fn func(expect, actual any) bool) {
	t.Helper()

	t.Run("type_mismatch", func(t *testing.T) {
		t.Parallel()

		if fn([]string{"a", "b", "c"}, []int{1, 2, 3}) {
			t.Fatalf("expected mismatch on different types")
		}
	})

	t.Run("same_order_and_permutation", func(t *testing.T) {
		t.Parallel()

		if !fn([]string{"a", "b", "c"}, []string{"b", "a", "c"}) {
			t.Fatalf("expected match for permutation")
		}

		if !fn([]int{1, 2, 3}, []int{1, 2, 3}) {
			t.Fatalf("expected match for identical slices")
		}

		if !fn([]int{1, 2, 3}, []int{1, 3, 2}) {
			t.Fatalf("expected match for permutation of ints")
		}

		if !fn([]any{1, 2, 3}, []any{1, 2, 3}) {
			t.Fatalf("expected match for identical any slice")
		}

		if !fn([]any{1, 2, 3}, []any{1, 3, 2}) {
			t.Fatalf("expected match for permutation of any")
		}
	})

	t.Run("missing_elements", func(t *testing.T) {
		t.Parallel()

		if fn([]int{1, 2, 3}, []int{1, 2}) {
			t.Fatalf("expected mismatch when actual shorter")
		}

		if fn([]any{1, 2, 3}, []any{1, 2}) {
			t.Fatalf("expected mismatch when actual shorter (any)")
		}
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
