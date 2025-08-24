package matcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestMatcherRecursionDepth(t *testing.T) {
	t.Parallel()

	t.Run("nested any matchers", func(t *testing.T) {
		t.Parallel()

		// Create deeply nested any matchers
		deepMatcher := createDeepNestedMatcher(3)
		candidate := map[string]any{
			"level3": map[string]any{
				"level2": map[string]any{
					"level1": map[string]any{
						"value": "found",
					},
				},
			},
		}

		// This should work without stack overflow
		result := Match(deepMatcher, candidate)
		require.True(t, result, "Deep nested matcher should match")
	})

	t.Run("very deep recursion", func(t *testing.T) {
		t.Parallel()

		// Test with very deep nesting (100 levels)
		deepMatcher := createDeepNestedMatcher(100)
		candidate := createDeepCandidate(100)

		// This should not cause stack overflow
		result := Match(deepMatcher, candidate)
		require.True(t, result, "Very deep nested matcher should match")
	})

	t.Run("mixed recursion types", func(t *testing.T) {
		t.Parallel()

		// Test mixed recursion with equals, contains, matches, and any
		matcher := Matcher{
			Any: []Matcher{
				{
					Equals: map[string]any{
						"type": "payment",
					},
					Any: []Matcher{
						{
							Contains: map[string]any{
								"amount": 100,
							},
							Any: []Matcher{
								{
									Matches: map[string]any{
										"currency": "^USD$",
									},
									Any: []Matcher{
										{
											Equals: map[string]any{
												"status": "approved",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		candidate := map[string]any{
			"type":     "payment",
			"amount":   100,
			"currency": "USD",
			"status":   "approved",
		}

		result := Match(matcher, candidate)
		require.True(t, result, "Mixed recursion matcher should match")
	})
}

// createDeepNestedMatcher creates a matcher with n levels of nested any.
func createDeepNestedMatcher(depth int) Matcher {
	if depth <= 0 {
		return Matcher{
			Equals: map[string]any{
				"value": "found",
			},
		}
	}

	return Matcher{
		Any: []Matcher{
			{
				Equals: map[string]any{
					"level" + string(rune('0'+depth)): map[string]any{},
				},
			},
			createDeepNestedMatcher(depth - 1),
		},
	}
}

// createDeepCandidate creates a candidate with n levels of nested maps.
func createDeepCandidate(depth int) map[string]any {
	if depth <= 0 {
		return map[string]any{
			"value": "found",
		}
	}

	return map[string]any{
		"level" + string(rune('0'+depth)): createDeepCandidate(depth - 1),
	}
}
