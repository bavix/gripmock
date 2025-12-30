package stuber_test

import (
	"testing"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// TestMatchStreamV2 - tests stream matching in V2.
//
//nolint:funlen
func TestMatchStreamV2(t *testing.T) {
	t.Parallel()
	// Clear all caches before test
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
			expected:   false, // Empty streams should not match - stub must have input conditions
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
			expected: false, // For bidirectional streaming, single message can match any stub item
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

			query := stuber.QueryV2{
				Service: "test",
				Method:  "test",
				Input:   tt.queryInput,
			}

			stub := &stuber.Stub{
				Service: "test",
				Method:  "test",
				Inputs:  tt.stubStream,
			}

			budgerigar := stuber.NewBudgerigar(features.New())
			budgerigar.PutMany(stub)

			result, err := budgerigar.FindByQueryV2(query)
			if err != nil {
				if tt.expected {
					t.Errorf("Expected match but got error: %v", err)
				}

				return
			}

			matched := result.Found() != nil
			if matched != tt.expected {
				t.Errorf("matchStreamV2() = %v, want %v", matched, tt.expected)
			}
		})
	}
}

// TestMatchWithStreamV2 - tests combined matching in V2.
func TestMatchWithStreamV2(t *testing.T) {
	t.Parallel()
	// Clear all caches before test
	stuber.ClearAllCaches()

	query := stuber.QueryV2{
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

	budgerigar := stuber.NewBudgerigar(features.New())
	budgerigar.PutMany(stub)

	result, err := budgerigar.FindByQueryV2(query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Found() == nil {
		t.Error("Expected match to return true for matching query and stub with stream")
	}

	nonMatchingQuery := stuber.QueryV2{
		Service: "test",
		Method:  "test",
		Input: []map[string]any{
			{"stream1": "different"},
		},
	}

	result, err = budgerigar.FindByQueryV2(nonMatchingQuery)
	if err != nil {
		return
	}

	if result.Found() != nil {
		t.Error("Expected match to return false for non-matching stream")
	}
}

// TestBackwardCompatibilityV2 - tests backward compatibility in V2.
func TestBackwardCompatibilityV2(t *testing.T) {
	t.Parallel()
	// Clear all caches before test
	stuber.ClearAllCaches()

	query := stuber.QueryV2{
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

	result, err := budgerigar.FindByQueryV2(query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Found() == nil {
		t.Error("Expected backward compatibility: single stream item should match against input")
	}
}

func TestV2UnaryRequest(t *testing.T) {
	t.Parallel()
	// Clear all caches before test
	stuber.ClearAllCaches()

	query := stuber.QueryV2{
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

	result, err := budgerigar.FindByQueryV2(query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Found() == nil {
		t.Error("Expected unary request to match stub input")
	}
}

func TestV2StreamRequest(t *testing.T) {
	t.Parallel()
	// Clear all caches before test
	stuber.ClearAllCaches()

	query := stuber.QueryV2{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"stream1": "value1"}, {"stream2": "value2"}},
	}

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"stream1": "value1"}},
			{Equals: map[string]any{"stream2": "value2"}},
		},
	}

	budgerigar := stuber.NewBudgerigar(features.New())
	budgerigar.PutMany(stub)

	result, err := budgerigar.FindByQueryV2(query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Found() == nil {
		t.Error("Expected stream request to match stub stream")
	}
}

func TestV2MultipleStreamsNoStubStream(t *testing.T) {
	t.Parallel()

	query := stuber.QueryV2{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}, {"key2": "value2"}},
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

	result, err := budgerigar.FindByQueryV2(query)
	if err != nil {
		return
	}

	if result.Found() != nil {
		t.Error("Expected no match for multiple streams without stream in stub")
	}
}

// TestV2Priority - tests priorities in V2.
func TestV2Priority(t *testing.T) {
	t.Parallel()
	// Clear all caches before test
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

	budgerigar := stuber.NewBudgerigar(features.New())
	budgerigar.PutMany(stub1, stub2)

	query := stuber.QueryV2{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}},
	}

	result, err := budgerigar.FindByQueryV2(query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Found() == nil {
		t.Error("Expected to find exact match")
	}

	if result.Found().Output.Data["result"] != "stub2" {
		t.Errorf("Expected to match higher priority stub, got %v", result.Found().Output.Data["result"])
	}
}
