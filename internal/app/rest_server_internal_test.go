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
	"github.com/stretchr/testify/suite"

	"github.com/bavix/features"
)

// mockExtender is a mock implementation of Extender for testing.
type mockExtender struct{}

func (m *mockExtender) Update(stubs []*stuber.Stub) error { return nil }
func (m *mockExtender) Wait(ctx context.Context)          {}

// RestServerTestSuite provides test suite for REST server functionality.
type RestServerTestSuite struct {
	suite.Suite

	server     *RestServer
	budgerigar *stuber.Budgerigar
}

// SetupSuite initializes the test suite.
func (s *RestServerTestSuite) SetupSuite() {
	s.budgerigar = stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(context.Background(), s.budgerigar, extender)
	s.Require().NoError(err)
	s.server = server
}

// SetupTest cleans up before each test.
func (s *RestServerTestSuite) SetupTest() {
	s.budgerigar.Clear()
}

// TestNewRestServer tests REST server creation.
func (s *RestServerTestSuite) TestNewRestServer() {
	ctx := context.Background()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}

	server, err := NewRestServer(ctx, budgerigar, extender)
	s.Require().NoError(err)
	s.Require().NotNil(server)
}

// TestAddStub tests stub addition functionality.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *RestServerTestSuite) TestAddStub() {
	tests := []struct {
		name           string
		jsonData       string
		expectedStatus int
		expectedError  error
	}{
		{
			name: "valid unary stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}},
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid client stream stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestClientStream",
				"inputs": [{"contains": {"key": "value"}}],
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid server stream stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestServerStream",
				"input": {"contains": {"key": "value"}},
				"output": {"stream": [{"result": "response"}]}
			}]`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid bidirectional stub",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestBidiStream",
				"inputs": [{"contains": {"key": "value"}}],
				"output": {"stream": [{"result": "response"}]}
			}]`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid stub - missing service",
			jsonData: `[{
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}},
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrServiceIsMissing,
		},
		{
			name: "invalid stub - missing method",
			jsonData: `[{
				"service": "test.Service",
				"input": {"contains": {"key": "value"}},
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrMethodIsMissing,
		},
		{
			name: "invalid unary stub - no input",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid unary stub - no output",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid client stream stub - no inputs",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestClientStream",
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			s.server.AddStub(w, req)

			s.Equal(tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				s.Contains(w.Body.String(), tt.expectedError.Error())
			}
		})
	}
}

// TestListStubs tests stub listing functionality.
func (s *RestServerTestSuite) TestListStubs() {
	req := httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
	w := httptest.NewRecorder()

	s.server.ListStubs(w, req)

	s.Equal(http.StatusOK, w.Code)

	var stubs []*stuber.Stub

	err := json.Unmarshal(w.Body.Bytes(), &stubs)
	s.Require().NoError(err)
	s.Require().NotNil(stubs)
}

// TestServicesList tests services listing functionality.
func (s *RestServerTestSuite) TestServicesList() {
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	w := httptest.NewRecorder()

	s.server.ServicesList(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.NotEmpty(w.Body.String())
}

// TestServiceMethodsList tests service methods listing functionality.
func (s *RestServerTestSuite) TestServiceMethodsList() {
	req := httptest.NewRequest(http.MethodGet, "/api/services/test.Service/methods", nil)
	w := httptest.NewRecorder()

	s.server.ServiceMethodsList(w, req, "test.Service")

	s.Equal(http.StatusOK, w.Code)
	s.NotEmpty(w.Body.String())
}

// TestDeleteStubByID tests stub deletion by ID.
func (s *RestServerTestSuite) TestDeleteStubByID() {
	randomUUID := uuid.New()
	randomID := randomUUID.String()
	req := httptest.NewRequest(http.MethodDelete, "/api/stubs/"+randomID, nil)
	w := httptest.NewRecorder()

	s.server.DeleteStubByID(w, req, randomUUID)

	s.Equal(http.StatusNoContent, w.Code)
}

// TestBatchStubsDelete tests batch stub deletion.
func (s *RestServerTestSuite) TestBatchStubsDelete() {
	requestBody := `["550e8400-e29b-41d4-a716-446655440000"]`
	req := httptest.NewRequest(http.MethodDelete, "/api/stubs", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.server.BatchStubsDelete(w, req)

	s.Equal(http.StatusOK, w.Code)
}

// TestListUsedStubs tests used stubs listing.
func (s *RestServerTestSuite) TestListUsedStubs() {
	req := httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
	w := httptest.NewRecorder()

	s.server.ListUsedStubs(w, req)

	s.Equal(http.StatusOK, w.Code)

	var stubs []*stuber.Stub

	err := json.Unmarshal(w.Body.Bytes(), &stubs)
	s.Require().NoError(err)
	s.Require().NotNil(stubs)
}

// TestListUnusedStubs tests unused stubs listing.
func (s *RestServerTestSuite) TestListUnusedStubs() {
	req := httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
	w := httptest.NewRecorder()

	s.server.ListUnusedStubs(w, req)

	s.Equal(http.StatusOK, w.Code)

	var stubs []*stuber.Stub

	err := json.Unmarshal(w.Body.Bytes(), &stubs)
	s.Require().NoError(err)
	s.Require().NotNil(stubs)
}

// TestPurgeStubs tests stub purging functionality.
func (s *RestServerTestSuite) TestPurgeStubs() {
	req := httptest.NewRequest(http.MethodDelete, "/api/stubs", nil)
	w := httptest.NewRecorder()

	s.server.PurgeStubs(w, req)

	s.Equal(http.StatusNoContent, w.Code)
}

// TestSearchStubs tests stub searching functionality.
func (s *RestServerTestSuite) TestSearchStubs() {
	searchData := `{"service": "test.Service", "method": "TestMethod", "data": {"key": "value"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.server.SearchStubs(w, req)

	// Should return 404 for non-existing stub
	s.Equal(http.StatusNotFound, w.Code)
}

// TestReadiness tests readiness endpoint.
func (s *RestServerTestSuite) TestReadiness() {
	req := httptest.NewRequest(http.MethodGet, "/api/health/readiness", nil)
	w := httptest.NewRecorder()

	s.server.Readiness(w, req)

	// Readiness can return either 200 or 503
	s.Require().True(w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable)
}

// TestLiveness tests liveness endpoint.
func (s *RestServerTestSuite) TestLiveness() {
	req := httptest.NewRequest(http.MethodGet, "/api/health/liveness", nil)
	w := httptest.NewRecorder()

	s.server.Liveness(w, req)

	s.Equal(http.StatusOK, w.Code)
}

// TestValidateStubIntegration tests stub validation integration.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *RestServerTestSuite) TestValidateStubIntegration() {
	tests := []struct {
		name      string
		stub      stuber.Stub
		errorType error
	}{
		{
			name: "valid unary stub",
			stub: stuber.Stub{
				Service: "test.Service",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
		},
		{
			name: "valid client stream stub",
			stub: stuber.Stub{
				Service: "test.Service",
				Method:  "TestClientStream",
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
		},
		{
			name: "valid server stream stub",
			stub: stuber.Stub{
				Service: "test.Service",
				Method:  "TestServerStream",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Stream: []any{map[string]any{"result": "response"}},
				},
			},
		},
		{
			name: "valid bidirectional stub",
			stub: stuber.Stub{
				Service: "test.Service",
				Method:  "TestBidiStream",
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Stream: []any{map[string]any{"result": "response"}},
				},
			},
		},
		{
			name: "missing service",
			stub: stuber.Stub{
				Method: "TestMethod",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			errorType: ErrServiceIsMissing,
		},
		{
			name: "missing method",
			stub: stuber.Stub{
				Service: "test.Service",
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			errorType: ErrMethodIsMissing,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := validateStub(&tt.stub)
			if tt.errorType != nil {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errorType.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// TestRestServerTestSuite runs the REST server test suite.
func TestRestServerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RestServerTestSuite))
}
