package stuber

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestStubExtendedFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stub     Stub
		expected string
	}{
		{
			name: "stub with times field",
			stub: Stub{
				ID:       uuid.New(),
				Service:  "test.Service",
				Method:   "TestMethod",
				Priority: 1,
				Times:    5,
				Input: InputData{
					Equals: map[string]any{"field": "value"},
				},
				Output: Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expected: `"times":5`,
		},
		{
			name: "stub with response headers",
			stub: Stub{
				ID:       uuid.New(),
				Service:  "test.Service",
				Method:   "TestMethod",
				Priority: 1,
				ResponseHeaders: map[string]string{
					"x-custom-header": "custom-value",
					"authorization":   "Bearer token",
				},
				Input: InputData{
					Equals: map[string]any{"field": "value"},
				},
				Output: Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expected: `"responseHeaders":{"authorization":"Bearer token","x-custom-header":"custom-value"}`,
		},
		{
			name: "stub with response trailers",
			stub: Stub{
				ID:       uuid.New(),
				Service:  "test.Service",
				Method:   "TestMethod",
				Priority: 1,
				ResponseTrailers: map[string]string{
					"x-status": "completed",
					"x-meta":   "trailer-data",
				},
				Input: InputData{
					Equals: map[string]any{"field": "value"},
				},
				Output: Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expected: `"responseTrailers":{"x-meta":"trailer-data","x-status":"completed"}`,
		},
		{
			name: "stub with all extended fields",
			stub: Stub{
				ID:       uuid.New(),
				Service:  "test.Service",
				Method:   "TestMethod",
				Priority: 1,
				Times:    10,
				ResponseHeaders: map[string]string{
					"x-request-id": "req-123",
				},
				ResponseTrailers: map[string]string{
					"x-response-time": "100ms",
				},
				Input: InputData{
					Equals: map[string]any{"field": "value"},
				},
				Output: Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expected: `"times":10`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test JSON marshaling
			data, err := json.Marshal(tt.stub)
			require.NoError(t, err)

			jsonStr := string(data)

			// Verify the expected field is present
			assert.Contains(t, jsonStr, tt.expected)

			// Test JSON unmarshaling
			var unmarshaled Stub

			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Verify the fields are correctly unmarshaled
			assert.Equal(t, tt.stub.Times, unmarshaled.Times)
			assert.Equal(t, tt.stub.ResponseHeaders, unmarshaled.ResponseHeaders)
			assert.Equal(t, tt.stub.ResponseTrailers, unmarshaled.ResponseTrailers)
		})
	}
}

func TestStubTimesFieldBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		times    int
		expected bool
	}{
		{"zero times means unlimited", 0, true},
		{"positive times is valid", 5, true},
		{"negative times should be handled", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := Stub{
				ID:       uuid.New(),
				Service:  "test.Service",
				Method:   "TestMethod",
				Priority: 1,
				Times:    tt.times,
				Input: InputData{
					Equals: map[string]any{"field": "value"},
				},
				Output: Output{
					Data: map[string]any{"result": "success"},
				},
			}

			// Test that the field can be marshaled and unmarshaled
			data, err := json.Marshal(stub)
			require.NoError(t, err)

			var unmarshaled Stub

			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.times, unmarshaled.Times)
		})
	}
}

//nolint:funlen
func TestStubResponseBehavior(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		fieldType        string
		responseData     map[string]string
		expectedContains string
	}{
		{
			name:         "empty headers",
			fieldType:    "headers",
			responseData: map[string]string{},
		},
		{
			name:      "single header",
			fieldType: "headers",
			responseData: map[string]string{
				"content-type": "application/json",
			},
			expectedContains: `"responseHeaders":{"content-type":"application/json"}`,
		},
		{
			name:      "multiple headers",
			fieldType: "headers",
			responseData: map[string]string{
				"content-type":    "application/json",
				"cache-control":   "no-cache",
				"x-custom-header": "custom-value",
			},
			expectedContains: `"responseHeaders"`,
		},
		{
			name:         "empty trailers",
			fieldType:    "trailers",
			responseData: map[string]string{},
		},
		{
			name:      "single trailer",
			fieldType: "trailers",
			responseData: map[string]string{
				"x-status": "completed",
			},
			expectedContains: `"responseTrailers":{"x-status":"completed"}`,
		},
		{
			name:      "multiple trailers",
			fieldType: "trailers",
			responseData: map[string]string{
				"x-status":        "completed",
				"x-response-time": "100ms",
				"x-meta":          "trailer-data",
			},
			expectedContains: `"x-status":"completed"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := Stub{
				ID:       uuid.New(),
				Service:  "test.Service",
				Method:   "TestMethod",
				Priority: 1,
				Input: InputData{
					Equals: map[string]any{"field": "value"},
				},
				Output: Output{
					Data: map[string]any{"result": "success"},
				},
			}

			const fieldTypeHeaders = "headers"
			if tc.fieldType == fieldTypeHeaders {
				stub.ResponseHeaders = tc.responseData
			} else {
				stub.ResponseTrailers = tc.responseData
			}

			data, err := json.Marshal(stub)
			require.NoError(t, err)

			if tc.expectedContains != "" {
				assert.Contains(t, string(data), tc.expectedContains)
			}

			var unmarshaled Stub

			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Handle nil vs empty map comparison
			//nolint:nestif
			if len(tc.responseData) == 0 {
				if tc.fieldType == fieldTypeHeaders {
					assert.True(t, len(unmarshaled.ResponseHeaders) == 0 || unmarshaled.ResponseHeaders == nil)
				} else {
					assert.True(t, len(unmarshaled.ResponseTrailers) == 0 || unmarshaled.ResponseTrailers == nil)
				}
			} else {
				if tc.fieldType == fieldTypeHeaders {
					assert.Equal(t, tc.responseData, unmarshaled.ResponseHeaders)
				} else {
					assert.Equal(t, tc.responseData, unmarshaled.ResponseTrailers)
				}
			}
		})
	}
}

func TestStubExtendedFieldsWithExistingFunctionality(t *testing.T) {
	t.Parallel()
	// Test that new fields don't break existing functionality
	stub := Stub{
		ID:       uuid.New(),
		Service:  "test.Service",
		Method:   "TestMethod",
		Priority: 5,
		Times:    3,
		ResponseHeaders: map[string]string{
			"x-request-id": "req-123",
		},
		ResponseTrailers: map[string]string{
			"x-status": "completed",
		},
		Headers: InputHeader{
			Equals: map[string]any{"authorization": "Bearer token"},
		},
		Input: InputData{
			Equals: map[string]any{"field": "value"},
		},
		Output: Output{
			Data: map[string]any{"result": "success"},
		},
	}

	// Test existing methods still work
	assert.True(t, stub.IsUnary())
	assert.False(t, stub.IsClientStream())
	assert.False(t, stub.IsServerStream())
	assert.False(t, stub.IsBidirectional())
	assert.Equal(t, stub.ID, stub.Key())
	assert.Equal(t, "test.Service", stub.Left())
	assert.Equal(t, "TestMethod", stub.Right())
	assert.Equal(t, 5, stub.Score())

	// Test JSON roundtrip
	data, err := json.Marshal(stub)
	require.NoError(t, err)

	var unmarshaled Stub

	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, stub.ID, unmarshaled.ID)
	assert.Equal(t, stub.Service, unmarshaled.Service)
	assert.Equal(t, stub.Method, unmarshaled.Method)
	assert.Equal(t, stub.Priority, unmarshaled.Priority)
	assert.Equal(t, stub.Times, unmarshaled.Times)
	assert.Equal(t, stub.ResponseHeaders, unmarshaled.ResponseHeaders)
	assert.Equal(t, stub.ResponseTrailers, unmarshaled.ResponseTrailers)
	assert.Equal(t, stub.Headers, unmarshaled.Headers)
	assert.Equal(t, stub.Input, unmarshaled.Input)
	assert.Equal(t, stub.Output, unmarshaled.Output)
}
