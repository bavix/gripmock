package app_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestDelayWithTypesDuration(t *testing.T) {
	t.Parallel()

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

	data, err := json.Marshal(stub)
	require.NoError(t, err)

	var unmarshaledStub stuber.Stub

	err = json.Unmarshal(data, &unmarshaledStub)
	require.NoError(t, err)

	require.Equal(t, types.Duration(100*time.Millisecond), unmarshaledStub.Output.Delay)
	require.Equal(t, 100*time.Millisecond, time.Duration(unmarshaledStub.Output.Delay))
}

func TestDelayStringFormat(t *testing.T) {
	t.Parallel()

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

	require.Equal(t, types.Duration(100*time.Millisecond), stub.Output.Delay)
	require.Equal(t, 100*time.Millisecond, time.Duration(stub.Output.Delay))
}
