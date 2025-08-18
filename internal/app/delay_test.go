package app_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gripmock/stuber"
	"github.com/gripmock/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelayWithTypesDuration(t *testing.T) {
	t.Parallel()

	// Test JSON marshaling/unmarshaling with delay using types.Duration
	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Input: stuber.InputData{
			Contains: map[string]any{"key": "value"},
		},
		Output: stuber.Output{
			Data:  map[string]any{"result": "success"},
			Delay: types.Duration(100 * time.Millisecond),
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(stub)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaledStub stuber.Stub

	err = json.Unmarshal(data, &unmarshaledStub)
	require.NoError(t, err)

	// Check that delay was preserved
	assert.Equal(t, types.Duration(100*time.Millisecond), unmarshaledStub.Output.Delay)
	assert.Equal(t, 100*time.Millisecond, time.Duration(unmarshaledStub.Output.Delay))
}

func TestDelayStringFormat(t *testing.T) {
	t.Parallel()

	// Test with string format (e.g., "100ms", "2.5s", "1h30m")
	jsonData := `{
		"service": "test.Service",
		"method": "TestMethod",
		"input": {"contains": {"key": "value"}},
		"output": {
			"data": {"result": "success"},
			"delay": "100ms"
		}
	}`

	var stub stuber.Stub

	err := json.Unmarshal([]byte(jsonData), &stub)
	require.NoError(t, err)

	assert.Equal(t, types.Duration(100*time.Millisecond), stub.Output.Delay)
	assert.Equal(t, 100*time.Millisecond, time.Duration(stub.Output.Delay))
}
