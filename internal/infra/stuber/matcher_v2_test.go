package stuber_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

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
			stubStream: []stuber.InputData{
				{Equals: map[string]any{"key1": "value1"}},
			},
			expected: true,
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
			stubStream: []stuber.InputData{
				{Equals: map[string]any{"key1": "different"}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			query := stuber.Query{
				Service: "test",
				Method:  "test",
				Input:   tt.queryInput,
			}

			stub := &stuber.Stub{
				Service: "test",
				Method:  "test",
				Inputs:  tt.stubStream,
			}

			budgerigar := stuber.NewBudgerigar()
			budgerigar.PutMany(stub)

			result, err := budgerigar.FindByQuery(query)
			if err != nil {
				if tt.expected {
					require.NoError(t, err, "Expected match but got error")
				}

				return
			}

			matched := result.Found() != nil
			require.Equal(t, tt.expected, matched, "matchStreamV2()")
		})
	}
}

func TestMatchWithStreamV2(t *testing.T) {
	t.Parallel()
	stuber.ClearAllCaches()

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input: []map[string]any{
			{"stream1": "value1"},
			{"stream2": "value2"},
		},
	}

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"stream1": "value1"}},
			{Equals: map[string]any{"stream2": "value2"}},
		},
	}

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(stub)

	result, err := budgerigar.FindByQuery(query)
	require.NoError(t, err)
	require.NotNil(t, result.Found(), "Expected match to return true for matching query and stub with stream")

	nonMatchingQuery := stuber.Query{
		Service: "test",
		Method:  "test",
		Input: []map[string]any{
			{"stream1": "different"},
		},
	}

	result, err = budgerigar.FindByQuery(nonMatchingQuery)
	require.NoError(t, err)
	require.Nil(t, result.Found(), "Expected match to return false for non-matching stream")
}

func TestV2MultipleStreamsSingleInputUsesLastElement(t *testing.T) {
	t.Parallel()

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}, {"key2": "value2"}},
	}

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			Equals: map[string]any{"key2": "value2"},
		},
	}

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(stub)

	result, err := budgerigar.FindByQuery(query)
	require.NoError(t, err)
	require.NotNil(t, result.Found(), "Expected match against last stream element for single-Input stub")
}

func TestV2Priority(t *testing.T) {
	t.Parallel()
	stuber.ClearAllCaches()

	stub1 := &stuber.Stub{
		Service:  "test",
		Method:   "test",
		Priority: 1,
		Input: stuber.InputData{
			Equals: map[string]any{"key1": "value1"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "stub1"},
		},
	}

	stub2 := &stuber.Stub{
		Service:  "test",
		Method:   "test",
		Priority: 2,
		Input: stuber.InputData{
			Equals: map[string]any{"key1": "value1"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "stub2"},
		},
	}

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(stub1, stub2)

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}},
	}

	result, err := budgerigar.FindByQuery(query)
	require.NoError(t, err)
	require.NotNil(t, result.Found(), "Expected to find exact match")
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
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"product_id": "PROD_789"}},
		},
		Output: stuber.Output{
			Data: map[string]any{"status": "processed"},
		},
	}

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(stub)

	// Two messages — exactly what grpctestify sends (no trailing empty; {} in gripmock
	// logs is an EOF artifact from the logging interceptor, not a real message).
	query := stuber.Query{
		Service: "ecommerce.EcommerceService",
		Method:  "SubmitProductReviews",
		Input: []map[string]any{
			{"product_id": "PROD_789", "rating": 5, "user_id": "USER_001"},
			{"product_id": "PROD_789", "rating": 4, "user_id": "USER_002"},
		},
	}

	result, err := budgerigar.FindByQuery(query)
	require.NoError(t, err)
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
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"product_id": "PROD_789"}},
		},
		Output: stuber.Output{
			Data: map[string]any{"status": "processed"},
		},
	}

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(stub)

	// Second message has wrong product_id
	query := stuber.Query{
		Service: "ecommerce.EcommerceService",
		Method:  "SubmitProductReviews",
		Input: []map[string]any{
			{"product_id": "PROD_789", "rating": 5},
			{"product_id": "PROD_WRONG", "rating": 4},
		},
	}

	result, err := budgerigar.FindByQuery(query)
	if err == nil {
		require.Nil(t, result.Found(), "Broadcast must fail if any message mismatches the pattern")
	}
}
