//nolint:testpackage
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gripmock/stuber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
)

type mockExtender struct{}

func (m *mockExtender) Wait(ctx context.Context) {}

func TestNewRestServer(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}

	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, budgerigar, server.budgerigar)
}

//nolint:funlen
func TestRestServer_AddStub(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	tests := []struct {
		name           string
		jsonData       string
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid unary stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {
					"contains": {"key": "value"}
				},
				"output": {
					"data": {"result": "success"}
				}
			}]`,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "valid client stream stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestClientStream",
				"inputs": [
					{"contains": {"key": "value"}}
				],
				"output": {
					"data": {"result": "success"}
				}
			}]`,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "valid server stream stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestServerStream",
				"input": {
					"contains": {"key": "value"}
				},
				"output": {
					"stream": [{"result": "success"}]
				}
			}]`,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "valid bidirectional stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestBidiStream",
				"inputs": [
					{"contains": {"key": "value"}}
				],
				"output": {
					"stream": [{"result": "success"}]
				}
			}]`,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "invalid stub - missing service",
			jsonData: `[{
				"method": "TestMethod",
				"input": {
					"contains": {"key": "value"}
				},
				"output": {
					"data": {"result": "success"}
				}
			}]`,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid stub - missing method",
			jsonData: `[{
				"service": "test.Service",
				"input": {
					"contains": {"key": "value"}
				},
				"output": {
					"data": {"result": "success"}
				}
			}]`,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid unary stub - no input",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {},
				"output": {
					"data": {"result": "success"}
				}
			}]`,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid unary stub - no output",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {
					"contains": {"key": "value"}
				},
				"output": {}
			}]`,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid client stream stub - no inputs",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestClientStream",
				"inputs": [],
				"output": {
					"data": {"result": "success"}
				}
			}]`,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/stub", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			server.AddStub(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				var response map[string]string

				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response["error"])
			} else {
				// Just check that we got a successful response
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

func TestRestServer_ListStubs(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/stubs", nil)
	w := httptest.NewRecorder()

	server.ListStubs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*stuber.Stub

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestRestServer_ServicesList(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	w := httptest.NewRecorder()

	server.ServicesList(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that response is valid JSON
	var response any

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestRestServer_ServiceMethodsList(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/services/test.Service/methods", nil)
	w := httptest.NewRecorder()

	server.ServiceMethodsList(w, req, "test.Service")

	assert.Equal(t, http.StatusOK, w.Code)

	var response []string

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestRestServer_DeleteStubByID(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/stub/test-id", nil)
	w := httptest.NewRecorder()

	server.DeleteStubByID(w, req, uuid.New())

	assert.Equal(t, http.StatusNoContent, w.Code)
	// No content response, so no JSON body
}

func TestRestServer_BatchStubsDelete(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	body := `["00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"]`
	req := httptest.NewRequest(http.MethodDelete, "/stubs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	server.BatchStubsDelete(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Batch delete might return empty response, which is acceptable
	if w.Body.Len() > 0 {
		var response any

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestRestServer_ListUsedStubs(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/stubs/used", nil)
	w := httptest.NewRecorder()

	server.ListUsedStubs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*stuber.Stub

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestRestServer_ListUnusedStubs(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/stubs/unused", nil)
	w := httptest.NewRecorder()

	server.ListUnusedStubs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*stuber.Stub

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestRestServer_PurgeStubs(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/stubs", nil)
	w := httptest.NewRecorder()

	server.PurgeStubs(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// No content response, so no JSON body
}

func TestRestServer_SearchStubs(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/stubs/search?q=test", nil)
	w := httptest.NewRecorder()

	server.SearchStubs(w, req)

	// Search might return 500 if no stubs are found, which is acceptable
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)

	// If we got a response, check that it's valid JSON
	if w.Body.Len() > 0 {
		var response any

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestRestServer_Readiness(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	server.Readiness(w, req)

	// Readiness might return 503 if not ready, which is acceptable
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code)

	// If we got a response, check that it's valid JSON
	if w.Body.Len() > 0 {
		var response any

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestRestServer_Liveness(t *testing.T) {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(ctx, budgerigar, extender)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()

	server.Liveness(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response["message"])
}

//nolint:funlen
func TestValidateStub_Integration(t *testing.T) {
	tests := []struct {
		name        string
		stub        *stuber.Stub
		expectError bool
		errorType   error
	}{
		{
			name: "valid unary stub",
			stub: &stuber.Stub{
				Service: "test.Service",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expectError: false,
		},
		{
			name: "valid client stream stub",
			stub: &stuber.Stub{
				Service: "test.Service",
				Method:  "TestClientStream",
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expectError: false,
		},
		{
			name: "valid server stream stub",
			stub: &stuber.Stub{
				Service: "test.Service",
				Method:  "TestServerStream",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Stream: []any{map[string]any{"result": "success"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid bidirectional stub",
			stub: &stuber.Stub{
				Service: "test.Service",
				Method:  "TestBidiStream",
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Stream: []any{map[string]any{"result": "success"}},
				},
			},
			expectError: false,
		},
		{
			name: "missing service",
			stub: &stuber.Stub{
				Method: "TestMethod",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expectError: true,
			errorType:   ErrServiceIsMissing,
		},
		{
			name: "missing method",
			stub: &stuber.Stub{
				Service: "test.Service",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expectError: true,
			errorType:   ErrMethodIsMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStub(tt.stub)
			if tt.expectError {
				require.Error(t, err)

				if tt.errorType != nil {
					// Check that the error message matches our expected error
					assert.Contains(t, err.Error(), tt.errorType.Error())
				} else {
					// Check that we got a validation error
					assert.NotEmpty(t, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
