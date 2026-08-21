package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestNewErrorFormatter(t *testing.T) {
	t.Parallel()

	formatter := NewErrorFormatter()
	require.NotNil(t, formatter)
}

func TestErrorFormatterFormatStubNotFoundError(t *testing.T) {
	t.Parallel()

	formatter := NewErrorFormatter()

	query := stuber.Query{
		Service: "test.Service",
		Method:  "TestMethod",
		Input:   []map[string]any{{"key": "value"}},
	}

	result := &stuber.Result{}

	err := formatter.FormatStubNotFoundError(query, result)
	require.Error(t, err)

	errorMsg := err.Error()
	require.Contains(t, errorMsg, "No matching stub found")
	require.Contains(t, errorMsg, "Service: test.Service")
	require.Contains(t, errorMsg, "Method: TestMethod")
	require.Contains(t, errorMsg, "Request input")
}
