package app

import (
	"testing"

	"github.com/gripmock/stuber"
	"github.com/stretchr/testify/require"
)

func TestNewErrorFormatter(t *testing.T) {
	t.Parallel()

	formatter := NewErrorFormatter()
	require.NotNil(t, formatter)
}

func TestErrorFormatter_CreateStubNotFoundError(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			err := formatter.CreateStubNotFoundError(tt.serviceName, tt.methodName, tt.details...)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestErrorFormatter_CreateClientStreamError(t *testing.T) {
	t.Parallel()

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
			err:         ErrServiceIsMissing,
			expected:    "Failed to find response for client stream (service: test.Service, method: TestMethod) - Error: service name is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := formatter.CreateClientStreamError(tt.serviceName, tt.methodName, tt.err)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestErrorFormatter_FormatStubNotFoundErrorV2(t *testing.T) {
	t.Parallel()

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
	require.Contains(t, errorMsg, "Can't find stub")
	require.Contains(t, errorMsg, "Service: test.Service")
	require.Contains(t, errorMsg, "Method: TestMethod")
	require.Contains(t, errorMsg, "Input")
}

func TestErrorFormatter_FormatStubNotFoundError(t *testing.T) {
	t.Parallel()

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
	require.Contains(t, errorMsg, "Can't find stub")
	require.Contains(t, errorMsg, "Service: test.Service")
	require.Contains(t, errorMsg, "Method: TestMethod")
	require.Contains(t, errorMsg, "Input")
}
