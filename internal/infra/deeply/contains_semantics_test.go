package deeply_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/deeply"
)

func TestContainsDoesNotSubstringMatchStrings(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		expect any
		actual any
		want   bool
	}{
		"whole string":             {"gripmock", "gripmock", true},
		"prefix of a longer value": {"grip", "gripmock", false},
		"suffix of a longer value": {"mock", "gripmock", false},
		"different case":           {"GripMock", "gripmock", false},
		"inside a field":           {map[string]any{"name": "grip"}, map[string]any{"name": "gripmock"}, false},
		"field matched whole":      {map[string]any{"name": "gripmock"}, map[string]any{"name": "gripmock"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.want,
				deeply.ContainsIgnoreArrayOrder(testCase.expect, testCase.actual))
		})
	}
}

func TestContainsIgnoresUndeclaredFields(t *testing.T) {
	t.Parallel()

	actual := map[string]any{
		"name":    "gripmock",
		"version": "3.20",
		"nested":  map[string]any{"a": 1, "b": 2},
	}

	require.True(t, deeply.ContainsIgnoreArrayOrder(map[string]any{"name": "gripmock"}, actual))
	require.True(t, deeply.ContainsIgnoreArrayOrder(map[string]any{"nested": map[string]any{"a": 1}}, actual),
		"objects are matched recursively as a subset")
	require.False(t, deeply.ContainsIgnoreArrayOrder(map[string]any{"nested": map[string]any{"a": 9}}, actual))
}
