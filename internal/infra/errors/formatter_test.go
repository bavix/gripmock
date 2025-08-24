package errors_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/errors"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestNewStubNotFoundFormatter(t *testing.T) {
	t.Parallel()

	formatter := errors.NewStubNotFoundFormatter()
	require.NotNil(t, formatter)
}

//nolint:funlen
func TestStubNotFoundFormatter_FormatV1(t *testing.T) {
	t.Parallel()

	formatter := errors.NewStubNotFoundFormatter()

	t.Run("without similar results", func(t *testing.T) {
		t.Parallel()

		query := stuber.Query{
			Service: "test.Service",
			Method:  "TestMethod",
			Data:    map[string]any{"key": "value"},
		}

		result := &stuber.Result{}

		err := formatter.FormatV1(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Can't find stub")
		require.Contains(t, errorMsg, "Service: test.Service")
		require.Contains(t, errorMsg, "Method: TestMethod")
		require.Contains(t, errorMsg, "Input")
		require.Contains(t, errorMsg, `"key": "value"`)
	})

	t.Run("with similar results", func(t *testing.T) {
		t.Parallel()

		query := stuber.Query{
			Service: "test.Service",
			Method:  "TestMethod",
			Data:    map[string]any{"key": "value"},
		}

		// Create a mock result that has similar stub
		result := &mockResult{
			similar: &stuber.Stub{
				Input: stuber.InputData{
					Equals:   map[string]any{"key": "similar_value"},
					Contains: map[string]any{"other": "data"},
					Matches:  map[string]any{"pattern": ".*"},
				},
			},
		}

		err := formatter.FormatV1(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Can't find stub")
		require.Contains(t, errorMsg, "Closest Match")
		require.Contains(t, errorMsg, "equals")
		require.Contains(t, errorMsg, "similar_value")
		require.Contains(t, errorMsg, "contains")
		require.Contains(t, errorMsg, "matches")
	})

	t.Run("with invalid JSON data", func(t *testing.T) {
		t.Parallel()

		query := stuber.Query{
			Service: "test.Service",
			Method:  "TestMethod",
			Data:    map[string]any{"func": func() {}}, // Invalid JSON
		}

		result := &stuber.Result{}

		err := formatter.FormatV1(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Error marshaling input")
	})
}

//nolint:funlen
func TestStubNotFoundFormatter_FormatV2(t *testing.T) {
	t.Parallel()

	formatter := errors.NewStubNotFoundFormatter()

	t.Run("single input", func(t *testing.T) {
		t.Parallel()

		query := stuber.QueryV2{
			Service: "test.Service",
			Method:  "TestMethod",
			Input: []map[string]any{
				{"key": "value"},
			},
		}

		result := &stuber.Result{}

		err := formatter.FormatV2(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Can't find stub")
		require.Contains(t, errorMsg, "Service: test.Service")
		require.Contains(t, errorMsg, "Method: TestMethod")
		require.Contains(t, errorMsg, "Input:")
		require.Contains(t, errorMsg, `"key": "value"`)
	})

	t.Run("multiple inputs (streaming)", func(t *testing.T) {
		t.Parallel()

		query := stuber.QueryV2{
			Service: "test.Service",
			Method:  "TestMethod",
			Input: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
		}

		result := &stuber.Result{}

		err := formatter.FormatV2(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Inputs:")
		require.Contains(t, errorMsg, "[0]")
		require.Contains(t, errorMsg, "[1]")
		require.Contains(t, errorMsg, "key1")
		require.Contains(t, errorMsg, "key2")
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		query := stuber.QueryV2{
			Service: "test.Service",
			Method:  "TestMethod",
			Input:   []map[string]any{},
		}

		result := &stuber.Result{}

		err := formatter.FormatV2(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Input: (empty)")
	})

	t.Run("with client streaming similar results", func(t *testing.T) {
		t.Parallel()

		query := stuber.QueryV2{
			Service: "test.Service",
			Method:  "TestMethod",
			Input: []map[string]any{
				{"key": "value"},
			},
		}

		stub := &stuber.Stub{
			Inputs: []stuber.InputData{
				{
					Equals:   map[string]any{"key": "similar1"},
					Contains: map[string]any{"other1": "data1"},
				},
				{
					Equals:   map[string]any{"key": "similar2"},
					Contains: map[string]any{"other2": "data2"},
				},
			},
		}

		result := &mockResult{
			similar: stub,
		}

		err := formatter.FormatV2(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Closest Match")
		require.Contains(t, errorMsg, "inputs[0]")
		require.Contains(t, errorMsg, "inputs[1]")
		require.Contains(t, errorMsg, "similar1")
		require.Contains(t, errorMsg, "similar2")
	})

	t.Run("with invalid JSON in streaming input", func(t *testing.T) {
		t.Parallel()

		query := stuber.QueryV2{
			Service: "test.Service",
			Method:  "TestMethod",
			Input: []map[string]any{
				{"valid": "data"},
				{"func": func() {}}, // Invalid JSON
			},
		}

		result := &stuber.Result{}

		err := formatter.FormatV2(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Error marshaling input")
	})
}

func TestStubNotFoundFormatter_FormatClosestMatches(t *testing.T) {
	t.Parallel()

	formatter := errors.NewStubNotFoundFormatter()

	t.Run("with invalid JSON in matches", func(t *testing.T) {
		t.Parallel()

		stub := &stuber.Stub{
			Input: stuber.InputData{
				Equals: map[string]any{"func": func() {}}, // Invalid JSON
			},
		}

		result := &mockResult{
			similar: stub,
		}

		query := stuber.QueryV2{
			Service: "test.Service",
			Method:  "TestMethod",
			Input:   []map[string]any{{"key": "value"}},
		}

		err := formatter.FormatV2(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.Contains(t, errorMsg, "Error marshaling match")
		require.Contains(t, errorMsg, "Raw match:")
	})

	t.Run("empty matches", func(t *testing.T) {
		t.Parallel()

		stub := &stuber.Stub{
			Input: stuber.InputData{
				Equals:   map[string]any{},
				Contains: map[string]any{},
				Matches:  map[string]any{},
			},
		}

		result := &mockResult{
			similar: stub,
		}

		query := stuber.QueryV2{
			Service: "test.Service",
			Method:  "TestMethod",
			Input:   []map[string]any{{"key": "value"}},
		}

		err := formatter.FormatV2(query, result)
		require.Error(t, err)

		errorMsg := err.Error()
		require.NotContains(t, errorMsg, "Closest Match")
	})
}

// Mock implementations for testing

type mockResult struct {
	found   *stuber.Stub
	similar *stuber.Stub
}

func (m *mockResult) Found() *stuber.Stub {
	return m.found
}

func (m *mockResult) Similar() *stuber.Stub {
	return m.similar
}
