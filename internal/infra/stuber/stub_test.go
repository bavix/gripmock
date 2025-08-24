package stuber //nolint:testpackage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestStub_Methods(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	stub := &Stub{
		ID:       id,
		Service:  "TestService",
		Method:   "TestMethod",
		Priority: 10,
	}

	require.Equal(t, id, stub.Key())
	require.Equal(t, "TestService", stub.Left())
	require.Equal(t, "TestMethod", stub.Right())
	require.Equal(t, 10, stub.Score())
}

func TestInputData_Methods(t *testing.T) {
	t.Parallel()

	inputData := InputData{
		IgnoreArrayOrder: true,
		Equals:           map[string]any{"key1": "value1"},
		Contains:         map[string]any{"key2": "value2"},
		Matches:          map[string]any{"key3": "value3"},
	}

	require.Equal(t, map[string]any{"key1": "value1"}, inputData.GetEquals())
	require.Equal(t, map[string]any{"key2": "value2"}, inputData.GetContains())
	require.Equal(t, map[string]any{"key3": "value3"}, inputData.GetMatches())
}

func TestInputHeader_Methods(t *testing.T) {
	t.Parallel()

	inputHeader := InputHeader{
		Equals:   map[string]any{"header1": "value1"},
		Contains: map[string]any{"header2": "value2"},
		Matches:  map[string]any{"header3": "value3"},
	}

	require.Equal(t, map[string]any{"header1": "value1"}, inputHeader.GetEquals())
	require.Equal(t, map[string]any{"header2": "value2"}, inputHeader.GetContains())
	require.Equal(t, map[string]any{"header3": "value3"}, inputHeader.GetMatches())
	require.Equal(t, 3, inputHeader.Len())
}

func TestInputHeader_Len_Empty(t *testing.T) {
	t.Parallel()

	inputHeader := InputHeader{}

	require.Equal(t, 0, inputHeader.Len())
}

func TestOutput_Fields(t *testing.T) {
	t.Parallel()

	code := codes.OK
	output := Output{
		Headers: map[string]string{"header1": "value1"},
		Data:    map[string]any{"data1": "value1"},
		Stream:  []any{"message1", "message2", "message3"},
		Error:   "test error",
		Code:    &code,
		Delay:   types.Duration(100),
	}

	require.Equal(t, map[string]string{"header1": "value1"}, output.Headers)
	require.Equal(t, map[string]any{"data1": "value1"}, output.Data)
	require.Equal(t, []any{"message1", "message2", "message3"}, output.Stream)
	require.Equal(t, "test error", output.Error)
	require.Equal(t, &code, output.Code)
	require.Equal(t, 100, int(output.Delay))
}

func TestOutput_Fields_EmptyStream(t *testing.T) {
	t.Parallel()

	output := Output{
		Headers: map[string]string{"header1": "value1"},
		Data:    map[string]any{"data1": "value1"},
		// Stream field is not set (should be nil)
	}

	require.Equal(t, map[string]string{"header1": "value1"}, output.Headers)
	require.Equal(t, map[string]any{"data1": "value1"}, output.Data)
	require.Nil(t, output.Stream)
}

func TestOutput_Fields_OptionalDelay(t *testing.T) {
	t.Parallel()

	output := Output{
		Headers: map[string]string{"header1": "value1"},
		Data:    map[string]any{"data1": "value1"},
		// Delay field is not set (should be zero value)
	}

	require.Equal(t, map[string]string{"header1": "value1"}, output.Headers)
	require.Equal(t, map[string]any{"data1": "value1"}, output.Data)
	require.Equal(t, types.Duration(0), output.Delay)
}

func TestOutput_Delay_JSONSerialization(t *testing.T) {
	t.Parallel()
	// Test JSON serialization with delay
	output := Output{
		Headers: map[string]string{"content-type": "application/json"},
		Data:    map[string]any{"message": "Hello World"},
		Delay:   types.Duration(100 * time.Millisecond),
	}

	jsonData, err := json.Marshal(output)
	require.NoError(t, err)
	require.Contains(t, string(jsonData), `"delay":"100ms"`)

	// Test JSON deserialization with delay
	var decodedOutput Output

	err = json.Unmarshal([]byte(`{"headers":{"content-type":"application/json"},"data":{"message":"Hello World"},"delay":"200ms"}`),
		&decodedOutput)
	require.NoError(t, err)
	require.Equal(t, types.Duration(200*time.Millisecond), decodedOutput.Delay)
}
