package stuber_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// findInFreshBudgerigar creates a new Budgerigar, registers stubs, and queries it.
func findInFreshBudgerigar(t *testing.T, query stuber.Query, stubs ...*stuber.Stub) *stuber.Result {
	t.Helper()

	b := stuber.NewBudgerigar()
	b.PutMany(stubs...)

	result, err := b.FindByQuery(query)
	require.NoError(t, err)

	return result
}

func TestFieldValueEqualsJsonNumber(t *testing.T) {
	t.Parallel()

	// Precision: same json.Number must match, different must not.
	num := json.Number("-773977811204288029")
	numSame := json.Number("-773977811204288029")
	numDiff := json.Number("-773977811204288000")

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input:   stuber.InputData{Equals: map[string]any{"id": num}},
	}

	b := stuber.NewBudgerigar()
	b.PutMany(stub)

	r1, err := b.FindByQuery(stuber.Query{Service: "test", Method: "test", Input: []map[string]any{{"id": numSame}}})
	require.NoError(t, err)
	require.NotNil(t, r1.Found(), "same json.Number values should match")

	r2, err := b.FindByQuery(stuber.Query{Service: "test", Method: "test", Input: []map[string]any{{"id": numDiff}}})
	require.NoError(t, err)
	require.Nil(t, r2.Found(), "different json.Number values should not match")
}

//nolint:funlen
func TestMatchData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		queryData map[string]any
		stubInput stuber.InputData
		expected  bool
	}{
		{
			name:      "empty data",
			queryData: map[string]any{},
			stubInput: stuber.InputData{},
			expected:  true,
		},
		{
			name:      "single element match",
			queryData: map[string]any{"key1": "value1"},
			stubInput: stuber.InputData{Equals: map[string]any{"key1": "value1"}},
			expected:  true,
		},
		{
			name:      "multiple elements match",
			queryData: map[string]any{"key1": "value1", "key2": "value2"},
			stubInput: stuber.InputData{Equals: map[string]any{"key1": "value1", "key2": "value2"}},
			expected:  true,
		},
		{
			name:      "element mismatch",
			queryData: map[string]any{"key1": "value1"},
			stubInput: stuber.InputData{Equals: map[string]any{"key1": "different"}},
			expected:  false,
		},
		{
			name:      "partial match with contains",
			queryData: map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"},
			stubInput: stuber.InputData{Contains: map[string]any{"key1": "value1", "key2": "value2"}},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			query := stuber.Query{Service: "test", Method: "test", Input: []map[string]any{tt.queryData}}
			stub := &stuber.Stub{Service: "test", Method: "test", Input: tt.stubInput}

			b := stuber.NewBudgerigar()
			b.PutMany(stub)

			result, err := b.FindByQuery(query)
			if err != nil {
				if tt.expected {
					require.NoError(t, err, "Expected match but got error")
				}

				return
			}

			require.Equal(t, tt.expected, result.Found() != nil, "matchData()")
		})
	}
}

//nolint:funlen
func TestMatchStreamV2(t *testing.T) {
	t.Parallel()
	stuber.ClearAllCaches()

	tests := []struct {
		name       string
		queryInput []map[string]any
		stubStream []stuber.InputData
		expected   bool
	}{
		{
			name:       "empty streams",
			queryInput: []map[string]any{},
			stubStream: []stuber.InputData{},
			expected:   false,
		},
		{
			name:       "single element match",
			queryInput: []map[string]any{{"key1": "value1"}},
			stubStream: []stuber.InputData{{Equals: map[string]any{"key1": "value1"}}},
			expected:   true,
		},
		{
			name:       "multiple elements match",
			queryInput: []map[string]any{{"key1": "value1"}, {"key2": "value2"}},
			stubStream: []stuber.InputData{
				{Equals: map[string]any{"key1": "value1"}},
				{Equals: map[string]any{"key2": "value2"}},
			},
			expected: true,
		},
		{
			name:       "length mismatch",
			queryInput: []map[string]any{{"key1": "value1"}},
			stubStream: []stuber.InputData{
				{Equals: map[string]any{"key1": "value1"}},
				{Equals: map[string]any{"key2": "value2"}},
			},
			expected: false,
		},
		{
			name:       "element mismatch",
			queryInput: []map[string]any{{"key1": "value1"}},
			stubStream: []stuber.InputData{{Equals: map[string]any{"key1": "different"}}},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			query := stuber.Query{Service: "test", Method: "test", Input: tt.queryInput}
			stub := &stuber.Stub{Service: "test", Method: "test", Inputs: tt.stubStream}

			b := stuber.NewBudgerigar()
			b.PutMany(stub)

			result, err := b.FindByQuery(query)
			if err != nil {
				if tt.expected {
					require.NoError(t, err, "Expected match but got error")
				}

				return
			}

			require.Equal(t, tt.expected, result.Found() != nil, "matchStreamV2()")
		})
	}
}

func TestV2MultipleStreamsSingleInputUsesLastElement(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input:   stuber.InputData{Equals: map[string]any{"key2": "value2"}},
	}

	result := findInFreshBudgerigar(t,
		stuber.Query{
			Service: "test",
			Method:  "test",
			Input:   []map[string]any{{"key1": "value1"}, {"key2": "value2"}},
		},
		stub,
	)
	require.NotNil(t, result.Found(), "Expected match against last stream element for single-Input stub")
}

func TestV2Priority(t *testing.T) {
	t.Parallel()
	stuber.ClearAllCaches()

	stub1 := &stuber.Stub{
		Service:  "test",
		Method:   "test",
		Priority: 1,
		Input:    stuber.InputData{Equals: map[string]any{"key1": "value1"}},
		Output:   stuber.Output{Data: map[string]any{"result": "stub1"}},
	}
	stub2 := &stuber.Stub{
		Service:  "test",
		Method:   "test",
		Priority: 2,
		Input:    stuber.InputData{Equals: map[string]any{"key1": "value1"}},
		Output:   stuber.Output{Data: map[string]any{"result": "stub2"}},
	}

	result := findInFreshBudgerigar(t,
		stuber.Query{Service: "test", Method: "test", Input: []map[string]any{{"key1": "value1"}}},
		stub1, stub2,
	)
	require.NotNil(t, result.Found())
	require.Equal(t, "stub2", result.Found().Output.Data["result"], "Expected to match higher priority stub")
}

// TestBroadcastInputsMatchesAllMessages verifies that a stub with a single inputs[0] pattern
// matches a client-streaming query with multiple messages (broadcast semantics).
// This reproduces the PROD_789 / case_client_streaming_simple scenario.
func TestBroadcastInputsMatchesAllMessages(t *testing.T) {
	t.Parallel()
	stuber.ClearAllCaches()

	stub := &stuber.Stub{
		Service: "ecommerce.EcommerceService",
		Method:  "SubmitProductReviews",
		Inputs:  []stuber.InputData{{Equals: map[string]any{"product_id": "PROD_789"}}},
		Output:  stuber.Output{Data: map[string]any{"status": "processed"}},
	}

	// Two messages — no trailing empty {}; the {} in gripmock logs is an EOF
	// artifact from the logging interceptor, not a real message.
	result := findInFreshBudgerigar(t,
		stuber.Query{
			Service: "ecommerce.EcommerceService",
			Method:  "SubmitProductReviews",
			Input: []map[string]any{
				{"product_id": "PROD_789", "rating": 5, "user_id": "USER_001"},
				{"product_id": "PROD_789", "rating": 4, "user_id": "USER_002"},
			},
		},
		stub,
	)
	require.NotNil(t, result.Found(), "Broadcast: single inputs pattern must match all messages")
	require.Equal(t, "processed", result.Found().Output.Data["status"])
}

// TestBroadcastInputsRejectsMismatch verifies that broadcast fails when any message
// does not match the single pattern.
func TestBroadcastInputsRejectsMismatch(t *testing.T) {
	t.Parallel()
	stuber.ClearAllCaches()

	stub := &stuber.Stub{
		Service: "ecommerce.EcommerceService",
		Method:  "SubmitProductReviews",
		Inputs:  []stuber.InputData{{Equals: map[string]any{"product_id": "PROD_789"}}},
		Output:  stuber.Output{Data: map[string]any{"status": "processed"}},
	}

	b := stuber.NewBudgerigar()
	b.PutMany(stub)

	result, err := b.FindByQuery(stuber.Query{
		Service: "ecommerce.EcommerceService",
		Method:  "SubmitProductReviews",
		Input: []map[string]any{
			{"product_id": "PROD_789", "rating": 5},
			{"product_id": "PROD_WRONG", "rating": 4},
		},
	})
	if err == nil {
		require.Nil(t, result.Found(), "Broadcast must fail if any message mismatches the pattern")
	}
}

func TestAnyOfInputMatching(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			Equals: map[string]any{"role": "vip"},
			AnyOf: []stuber.AnyOfElement{
				{Equals: map[string]any{"name": "Alice"}},
				{Matches: map[string]any{"name": "^admin_"}},
			},
		},
	}

	b := stuber.NewBudgerigar()
	b.PutMany(stub)

	cases := []struct {
		name    string
		input   map[string]any
		wantHit bool
	}{
		{"alice+vip matches", map[string]any{"role": "vip", "name": "Alice"}, true},
		{"admin+vip matches", map[string]any{"role": "vip", "name": "admin_john"}, true},
		{"alice+user no match", map[string]any{"role": "user", "name": "Alice"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := b.FindByQuery(stuber.Query{
				Service: "test",
				Method:  "test",
				Input:   []map[string]any{tc.input},
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantHit, result.Found() != nil)
		})
	}
}

func TestAnyOfIgnoreArrayOrderScopedPerAlternative(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			IgnoreArrayOrder: false,
			AnyOf: []stuber.AnyOfElement{
				{
					IgnoreArrayOrder: true,
					Equals:           map[string]any{"items": []any{"a", "b"}},
				},
			},
		},
	}

	result := findInFreshBudgerigar(t,
		stuber.Query{Service: "test", Method: "test", Input: []map[string]any{{"items": []any{"b", "a"}}}},
		stub,
	)
	require.NotNil(t, result.Found(), "anyOf alternative must use its own ignoreArrayOrder")
}
