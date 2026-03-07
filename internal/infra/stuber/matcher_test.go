package stuber_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestFieldValueEqualsJsonNumber(t *testing.T) {
	t.Parallel()

	// Test that json.Number comparison works for large integers
	// This is critical for UUID high/low matching where precision matters
	num1 := json.Number("-773977811204288029")
	num2 := json.Number("-773977811204288029")
	num3 := json.Number("-773977811204288000") // Different value (precision loss)

	// Create stub with json.Number
	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			Equals: map[string]any{"id": num1},
		},
	}

	budgerigar := stuber.NewBudgerigar(features.New())
	budgerigar.PutMany(stub)

	// Query with same json.Number value should match
	query1 := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"id": num2}},
	}
	result1, err1 := budgerigar.FindByQuery(query1)
	require.NoError(t, err1)
	require.NotNil(t, result1.Found(), "same json.Number values should match")

	// Query with different json.Number value should not match
	query2 := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"id": num3}},
	}
	result2, err2 := budgerigar.FindByQuery(query2)
	require.NoError(t, err2)
	require.Nil(t, result2.Found(), "different json.Number values should not match")
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
			// Test through public API
			query := stuber.Query{
				Service: "test",
				Method:  "test",
				Input:   []map[string]any{tt.queryData},
			}

			stub := &stuber.Stub{
				Service: "test",
				Method:  "test",
				Input:   tt.stubInput,
			}

			budgerigar := stuber.NewBudgerigar(features.New())
			budgerigar.PutMany(stub)

			result, err := budgerigar.FindByQuery(query)
			if err != nil {
				if tt.expected {
					require.NoError(t, err, "Expected match but got error")
				}

				return
			}

			matched := result.Found() != nil
			require.Equal(t, tt.expected, matched, "matchData()")
		})
	}
}

func TestMatchWithData(t *testing.T) {
	t.Parallel()

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"name": "John", "age": 30}},
	}

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			Equals: map[string]any{"name": "John", "age": 30},
		},
	}

	budgerigar := stuber.NewBudgerigar(features.New())
	budgerigar.PutMany(stub)

	result, err := budgerigar.FindByQuery(query)
	require.NoError(t, err)
	require.NotNil(t, result.Found(), "Expected match to return true for matching query and stub with data")

	nonMatchingQuery := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"name": "John", "age": 25}}, // Different age
	}

	result, err = budgerigar.FindByQuery(nonMatchingQuery)
	require.NoError(t, err)
	require.Nil(t, result.Found(), "Expected match to return false for non-matching data")
}

func TestBackwardCompatibility(t *testing.T) {
	t.Parallel()

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}},
	}

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			Equals: map[string]any{"key1": "value1"},
		},
	}

	budgerigar := stuber.NewBudgerigar(features.New())
	budgerigar.PutMany(stub)

	result, err := budgerigar.FindByQuery(query)
	require.NoError(t, err)
	require.NotNil(t, result.Found(), "Expected backward compatibility: input should match against stub")
}
