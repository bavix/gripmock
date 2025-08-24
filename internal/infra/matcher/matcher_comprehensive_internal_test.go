package matcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMatcherComprehensive demonstrates all matcher capabilities.
//

//nolint:funlen,maintidx
func TestMatcherComprehensive(t *testing.T) {
	t.Parallel()

	t.Run("AND by default - equals, contains, matches", func(t *testing.T) {
		t.Parallel()

		// All conditions must be satisfied (AND logic)
		m := Matcher{
			Equals: map[string]any{
				"user_id": "123",
				"status":  "active",
			},
			Contains: map[string]any{
				"message": "hello",
			},
			Matches: map[string]string{
				"email": "^[a-z]+@example\\.com$",
			},
		}

		// Should match - all conditions satisfied
		candidate := map[string]any{
			"user_id": "123",
			"status":  "active",
			"message": "hello world",
			"email":   "john@example.com",
		}
		require.True(t, Match(m, candidate), "Should match when all conditions are satisfied")

		// Should not match - missing equals condition
		candidate2 := map[string]any{
			"user_id": "123",
			"status":  "inactive", // Different status
			"message": "hello world",
			"email":   "john@example.com",
		}
		require.False(t, Match(m, candidate2), "Should not match when equals condition fails")

		// Should not match - missing contains condition
		candidate3 := map[string]any{
			"user_id": "123",
			"status":  "active",
			"message": "goodbye world", // Does not contain "hello"
			"email":   "john@example.com",
		}
		require.False(t, Match(m, candidate3), "Should not match when contains condition fails")

		// Should not match - missing matches condition
		candidate4 := map[string]any{
			"user_id": "123",
			"status":  "active",
			"message": "hello world",
			"email":   "invalid-email", // Does not match regex
		}
		require.False(t, Match(m, candidate4), "Should not match when matches condition fails")
	})

	t.Run("OR logic with any", func(t *testing.T) {
		t.Parallel()

		// Any of the nested matchers can match (OR logic)
		m := Matcher{
			Any: []Matcher{
				{
					Equals: map[string]any{
						"type": "user",
						"id":   "123",
					},
				},
				{
					Equals: map[string]any{
						"type": "admin",
						"id":   "456",
					},
				},
				{
					Contains: map[string]any{
						"message": "error",
					},
				},
			},
		}

		// Should match - first condition
		candidate1 := map[string]any{
			"type": "user",
			"id":   "123",
		}
		require.True(t, Match(m, candidate1), "Should match first any condition")

		// Should match - second condition
		candidate2 := map[string]any{
			"type": "admin",
			"id":   "456",
		}
		require.True(t, Match(m, candidate2), "Should match second any condition")

		// Should match - third condition
		candidate3 := map[string]any{
			"message": "system error occurred",
		}
		require.True(t, Match(m, candidate3), "Should match third any condition")

		// Should not match - none of the conditions
		candidate4 := map[string]any{
			"type": "guest",
			"id":   "789",
		}
		require.False(t, Match(m, candidate4), "Should not match when none of any conditions are satisfied")
	})

	t.Run("ignoreArrayOrder for arrays", func(t *testing.T) {
		t.Parallel()

		t.Run("with ignoreArrayOrder=true", func(t *testing.T) {
			t.Parallel()

			m := Matcher{
				Equals: map[string]any{
					"tags": []any{"a", "b", "c"},
				},
				IgnoreArrayOrder: true,
			}

			// Should match - same elements, different order
			candidate1 := map[string]any{
				"tags": []any{"c", "a", "b"},
			}
			require.True(t, Match(m, candidate1), "Should match arrays with different order when ignoreArrayOrder=true")

			// Should match - same elements, same order
			candidate2 := map[string]any{
				"tags": []any{"a", "b", "c"},
			}
			require.True(t, Match(m, candidate2), "Should match arrays with same order when ignoreArrayOrder=true")

			// Should not match - different elements
			candidate3 := map[string]any{
				"tags": []any{"a", "b", "d"},
			}
			require.False(t, Match(m, candidate3), "Should not match arrays with different elements")

			// Should not match - different length
			candidate4 := map[string]any{
				"tags": []any{"a", "b"},
			}
			require.False(t, Match(m, candidate4), "Should not match arrays with different length")
		})

		t.Run("with ignoreArrayOrder=false", func(t *testing.T) {
			t.Parallel()

			m := Matcher{
				Equals: map[string]any{
					"tags": []any{"a", "b", "c"},
				},
				IgnoreArrayOrder: false,
			}

			// Should not match - different order
			candidate1 := map[string]any{
				"tags": []any{"c", "a", "b"},
			}
			require.False(t, Match(m, candidate1), "Should not match arrays with different order when ignoreArrayOrder=false")

			// Should match - same order
			candidate2 := map[string]any{
				"tags": []any{"a", "b", "c"},
			}
			require.True(t, Match(m, candidate2), "Should match arrays with same order when ignoreArrayOrder=false")
		})

		t.Run("nested arrays with ignoreArrayOrder", func(t *testing.T) {
			t.Parallel()

			m := Matcher{
				Equals: map[string]any{
					"users": []any{
						map[string]any{"id": "1", "name": "Alice"},
						map[string]any{"id": "2", "name": "Bob"},
					},
				},
				IgnoreArrayOrder: true,
			}

			// Should match - same objects, different order
			candidate1 := map[string]any{
				"users": []any{
					map[string]any{"id": "2", "name": "Bob"},
					map[string]any{"id": "1", "name": "Alice"},
				},
			}
			require.True(t, Match(m, candidate1), "Should match nested arrays with different order")

			// Should not match - different objects
			candidate2 := map[string]any{
				"users": []any{
					map[string]any{"id": "1", "name": "Alice"},
					map[string]any{"id": "3", "name": "Charlie"},
				},
			}
			require.False(t, Match(m, candidate2), "Should not match nested arrays with different objects")
		})
	})

	t.Run("complex nested structures", func(t *testing.T) {
		t.Parallel()

		m := Matcher{
			Equals: map[string]any{
				"user": map[string]any{
					"id":   "123",
					"name": "John",
					"roles": []any{
						"admin",
						"user",
					},
				},
			},
			Contains: map[string]any{
				"metadata": map[string]any{
					"tags": []any{"important"},
				},
			},
			Matches: map[string]string{
				"email": "^[a-z]+@example\\.com$",
			},
			IgnoreArrayOrder: true,
		}

		// Should match - all conditions satisfied
		candidate := map[string]any{
			"user": map[string]any{
				"id":   "123",
				"name": "John",
				"roles": []any{
					"user",
					"admin", // Different order, but IgnoreArrayOrder=true
				},
			},
			"metadata": map[string]any{
				"tags": []any{"important", "urgent"},
			},
			"email": "john@example.com",
		}
		require.True(t, Match(m, candidate), "Should match complex nested structure")
	})

	t.Run("legacy stub compatibility", func(t *testing.T) {
		t.Parallel()

		// Simulate legacy stub format
		legacyMatcher := Matcher{
			Equals: map[string]any{
				"method": "GET",
				"path":   "/api/users",
			},
		}

		// Should match legacy format
		legacyCandidate := map[string]any{
			"method": "GET",
			"path":   "/api/users",
			"headers": map[string]any{
				"content-type": "application/json",
			},
		}
		require.True(t, Match(legacyMatcher, legacyCandidate), "Should match legacy stub format")
	})

	t.Run("v4 stub features", func(t *testing.T) {
		t.Parallel()

		// Simulate v4 stub format with advanced features
		v4Matcher := Matcher{
			Equals: map[string]any{
				"service": "payments",
				"method":  "ProcessPayment",
			},
			Contains: map[string]any{
				"request": map[string]any{
					"amount": 100.0,
				},
			},
			Matches: map[string]string{
				"user_id": "^user_[0-9]+$",
			},
			Any: []Matcher{
				{
					Equals: map[string]any{
						"currency": "USD",
					},
				},
				{
					Equals: map[string]any{
						"currency": "EUR",
					},
				},
			},
			IgnoreArrayOrder: true,
		}

		// Should match v4 format with USD
		v4CandidateUSD := map[string]any{
			"service": "payments",
			"method":  "ProcessPayment",
			"request": map[string]any{
				"amount": 100.0,
				"items":  []any{"item1", "item2"},
			},
			"user_id":  "user_123",
			"currency": "USD",
		}
		require.True(t, Match(v4Matcher, v4CandidateUSD), "Should match v4 stub format with USD")

		// Should match v4 format with EUR
		v4CandidateEUR := map[string]any{
			"service": "payments",
			"method":  "ProcessPayment",
			"request": map[string]any{
				"amount": 100.0,
				"items":  []any{"item2", "item1"}, // Different order, but IgnoreArrayOrder=true
			},
			"user_id":  "user_456",
			"currency": "EUR",
		}
		require.True(t, Match(v4Matcher, v4CandidateEUR), "Should match v4 stub format with EUR")

		// Should not match - invalid user_id format
		v4CandidateInvalid := map[string]any{
			"service": "payments",
			"method":  "ProcessPayment",
			"request": map[string]any{
				"amount": 100.0,
			},
			"user_id":  "invalid_user_id", // Does not match regex
			"currency": "USD",
		}
		require.False(t, Match(v4Matcher, v4CandidateInvalid), "Should not match v4 stub with invalid user_id")
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Parallel()

		t.Run("empty matcher", func(t *testing.T) {
			t.Parallel()

			m := Matcher{}
			candidate := map[string]any{"any": "value"}
			require.True(t, Match(m, candidate), "Empty matcher should match any candidate")
		})

		t.Run("empty candidate", func(t *testing.T) {
			t.Parallel()

			m := Matcher{
				Equals: map[string]any{"key": "value"},
			}
			candidate := map[string]any{}
			require.False(t, Match(m, candidate), "Empty candidate should not match non-empty matcher")
		})

		t.Run("nil values", func(t *testing.T) {
			t.Parallel()

			m := Matcher{
				Equals: map[string]any{"key": nil},
			}
			candidate := map[string]any{"key": nil}
			require.True(t, Match(m, candidate), "Nil values should match")

			candidate2 := map[string]any{"key": "value"}
			require.False(t, Match(m, candidate2), "Non-nil value should not match nil matcher")
		})

		t.Run("numeric types", func(t *testing.T) {
			t.Parallel()

			m := Matcher{
				Equals: map[string]any{
					"int":    42,
					"float":  3.14,
					"string": "42",
				},
			}

			// Should match - same numeric values
			candidate := map[string]any{
				"int":    42,
				"float":  3.14,
				"string": "42",
			}
			require.True(t, Match(m, candidate), "Should match numeric values")

			// Should not match - different numeric types
			candidate2 := map[string]any{
				"int":    42.0, // Float instead of int
				"float":  3.14,
				"string": "42",
			}
			require.False(t, Match(m, candidate2), "Should not match different numeric types")
		})
	})
}
