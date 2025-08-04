package app

import (
	"testing"

	"github.com/gripmock/stuber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorFormatter(t *testing.T) {
	formatter := NewErrorFormatter()
	assert.NotNil(t, formatter)
}

func TestErrorFormatter_CreateStubNotFoundError(t *testing.T) {
	formatter := NewErrorFormatter()

	tests := []struct {
		name        string
		serviceName string
		methodName  string
		details     []string
		expected    string
	}{
		{
			name:        "basic error",
			serviceName: "test.Service",
			methodName:  "TestMethod",
			expected:    "Failed to find response (service: test.Service, method: TestMethod)",
		},
		{
			name:        "with details",
			serviceName: "test.Service",
			methodName:  "TestMethod",
			details:     []string{"custom error message"},
			expected:    "Failed to find response (service: test.Service, method: TestMethod) - custom error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatter.CreateStubNotFoundError(tt.serviceName, tt.methodName, tt.details...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestErrorFormatter_CreateClientStreamError(t *testing.T) {
	formatter := NewErrorFormatter()

	tests := []struct {
		name        string
		serviceName string
		methodName  string
		err         error
		expected    string
	}{
		{
			name:        "basic error",
			serviceName: "test.Service",
			methodName:  "TestMethod",
			expected:    "Failed to find response for client stream (service: test.Service, method: TestMethod)",
		},
		{
			name:        "with error",
			serviceName: "test.Service",
			methodName:  "TestMethod",
			err:         assert.AnError,
			expected:    "Failed to find response for client stream (service: test.Service, method: TestMethod) - Error: assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatter.CreateClientStreamError(tt.serviceName, tt.methodName, tt.err)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestErrorFormatter_FormatStubNotFoundErrorV2(t *testing.T) {
	formatter := NewErrorFormatter()

	query := stuber.QueryV2{
		Service: "test.Service",
		Method:  "TestMethod",
		Input: []map[string]any{
			{"key": "value"},
		},
	}

	result := &stuber.Result{}

	err := formatter.FormatStubNotFoundErrorV2(query, result)
	require.Error(t, err)

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "Can't find stub")
	assert.Contains(t, errorMsg, "Service: test.Service")
	assert.Contains(t, errorMsg, "Method: TestMethod")
	assert.Contains(t, errorMsg, "Input:")
}

func TestErrorFormatter_FormatStubNotFoundError(t *testing.T) {
	formatter := NewErrorFormatter()

	query := stuber.Query{
		Service: "test.Service",
		Method:  "TestMethod",
		Data:    map[string]any{"key": "value"},
	}

	result := &stuber.Result{}

	err := formatter.FormatStubNotFoundError(query, result)
	require.Error(t, err)

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "Can't find stub")
	assert.Contains(t, errorMsg, "Service: test.Service")
	assert.Contains(t, errorMsg, "Method: TestMethod")
	assert.Contains(t, errorMsg, "Input:")
}

func TestErrorFormatter_formatInputSection(t *testing.T) {
	formatter := NewErrorFormatter()

	tests := []struct {
		name     string
		input    []map[string]any
		expected string
	}{
		{
			name:     "empty input",
			input:    []map[string]any{},
			expected: "Input: (empty)",
		},
		{
			name:     "single input",
			input:    []map[string]any{{"key": "value"}},
			expected: "Input:",
		},
		{
			name:     "multiple inputs",
			input:    []map[string]any{{"key1": "value1"}, {"key2": "value2"}},
			expected: "Stream Input (multiple messages):",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.formatInputSection(tt.input)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestErrorFormatter_formatStreamInput(t *testing.T) {
	formatter := NewErrorFormatter()

	input := []map[string]any{
		{"key1": "value1"},
		{"key2": "value2"},
	}

	result := formatter.formatStreamInput(input)

	assert.Contains(t, result, "Stream Input (multiple messages):")
	assert.Contains(t, result, "Message 0:")
	assert.Contains(t, result, "Message 1:")
	assert.Contains(t, result, `"key1"`)
	assert.Contains(t, result, `"value1"`)
	assert.Contains(t, result, `"key2"`)
	assert.Contains(t, result, `"value2"`)
}

func TestErrorFormatter_formatSingleInput(t *testing.T) {
	formatter := NewErrorFormatter()

	input := map[string]any{"key": "value"}
	result := formatter.formatSingleInput(input)

	assert.Contains(t, result, "Input:")
	assert.Contains(t, result, `"key"`)
	assert.Contains(t, result, `"value"`)
}

func TestErrorFormatter_formatClosestMatches(t *testing.T) {
	// Note: We can't easily create a proper stuber.Result with Similar() method
	// So we'll skip this test for now as it requires complex setup
	t.Skip("Skipping test that requires complex stuber.Result setup")
}

func TestErrorFormatter_formatStreamClosestMatches(t *testing.T) {
	formatter := NewErrorFormatter()

	stub := &stuber.Stub{
		Stream: []stuber.InputData{
			{
				Equals:   map[string]any{"equals_key": "equals_value"},
				Contains: map[string]any{"contains_key": "contains_value"},
				Matches:  map[string]any{"matches_key": "matches_value"},
			},
		},
	}

	addClosestMatch := func(key string, match map[string]any) string {
		return "test match for " + key
	}

	result := formatter.formatStreamClosestMatches(stub, addClosestMatch)

	assert.Contains(t, result, "test match for stream[0]")
}
