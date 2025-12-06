package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
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
				"method": "TestBidirectional",
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
		},
		{
			name: "invalid stub - missing method",
			jsonData: `[{
				"service": "test.Service",
				"input": {"contains": {"key": "value"}},
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
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
			req := httptest.NewRequest(http.MethodPost, "/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			s.server.AddStub(w, req)

			s.Equal(tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				// AddStub returns array of UUIDs
				var response []string

				err := json.Unmarshal(w.Body.Bytes(), &response)
				s.Require().NoError(err)
				s.NotEmpty(response)
				// Check that it's a valid UUID
				_, err = uuid.Parse(response[0])
				s.Require().NoError(err)
			}
		})
	}
}

// TestDeleteStubByID tests stub deletion by ID.
func (s *RestServerTestSuite) TestDeleteStubByID() {
	// Add a stub first
	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Input: stuber.InputData{
			Contains: map[string]any{"key": "value"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "success"},
		},
	}

	s.budgerigar.PutMany(stub)

	// Get the stub ID
	stubs := s.budgerigar.All()
	s.Require().NotEmpty(stubs)
	stubID := stubs[0].ID

	// Delete the stub
	w := httptest.NewRecorder()
	s.server.DeleteStubByID(w, nil, stubID)

	s.Equal(http.StatusNoContent, w.Code)

	// Verify the stub was deleted
	stubs = s.budgerigar.All()
	s.Empty(stubs)
}

// TestBatchStubsDelete tests batch stub deletion.
func (s *RestServerTestSuite) TestBatchStubsDelete() {
	// Add multiple stubs
	stub1 := &stuber.Stub{
		Service: "test.Service1",
		Method:  "TestMethod1",
		Input: stuber.InputData{
			Contains: map[string]any{"key": "value1"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "success1"},
		},
	}

	stub2 := &stuber.Stub{
		Service: "test.Service2",
		Method:  "TestMethod2",
		Input: stuber.InputData{
			Contains: map[string]any{"key": "value2"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "success2"},
		},
	}

	s.budgerigar.PutMany(stub1, stub2)

	// Get stub IDs
	stubs := s.budgerigar.All()
	s.Require().Len(stubs, 2)

	stubIDs := []uuid.UUID{stubs[0].ID, stubs[1].ID}
	jsonData, err := json.Marshal(stubIDs)
	s.Require().NoError(err)

	// Delete stubs in batch
	req := httptest.NewRequest(http.MethodPost, "/stubs/batchDelete", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.server.BatchStubsDelete(w, req)

	s.Equal(http.StatusOK, w.Code)

	// Verify stubs were deleted
	stubs = s.budgerigar.All()
	s.Empty(stubs)
}

// TestListStubs tests listing all stubs.
func (s *RestServerTestSuite) TestListStubs() {
	// Add a stub
	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Input: stuber.InputData{
			Contains: map[string]any{"key": "value"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "success"},
		},
	}

	s.budgerigar.PutMany(stub)

	// List stubs
	w := httptest.NewRecorder()
	s.server.ListStubs(w, nil)

	s.Equal(http.StatusOK, w.Code)

	// ListStubs returns array of stubs
	var response []*stuber.Stub

	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	s.Len(response, 1)
}

// TestListUnusedStubs tests listing unused stubs.
func (s *RestServerTestSuite) TestListUnusedStubs() {
	w := httptest.NewRecorder()
	s.server.ListUnusedStubs(w, nil)

	s.Equal(http.StatusOK, w.Code)

	// ListUnusedStubs returns array of stubs
	var response []*stuber.Stub

	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	s.Empty(response) // No stubs added yet
}

// TestListUsedStubs tests listing used stubs.
func (s *RestServerTestSuite) TestListUsedStubs() {
	w := httptest.NewRecorder()
	s.server.ListUsedStubs(w, nil)

	s.Equal(http.StatusOK, w.Code)

	// ListUsedStubs returns array of stubs
	var response []*stuber.Stub

	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	s.Empty(response) // No stubs used yet
}

// TestLiveness tests liveness endpoint.
func (s *RestServerTestSuite) TestLiveness() {
	w := httptest.NewRecorder()
	s.server.Liveness(w, nil)

	s.Equal(http.StatusOK, w.Code)
}

// TestReadiness tests readiness endpoint.
func (s *RestServerTestSuite) TestReadiness() {
	// Wait for server to be ready with timeout
	timeout := time.After(2 * time.Second)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			s.Fail("Server did not become ready within timeout")

			return
		case <-ticker.C:
			w := httptest.NewRecorder()
			s.server.Readiness(w, nil)

			if w.Code == http.StatusOK {
				// Server is ready, final check
				w := httptest.NewRecorder()
				s.server.Readiness(w, nil)
				s.Equal(http.StatusOK, w.Code)

				return
			}
		}
	}
}

// TestPurgeStubs tests purging all stubs.
func (s *RestServerTestSuite) TestPurgeStubs() {
	// Add a stub
	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Input: stuber.InputData{
			Contains: map[string]any{"key": "value"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "success"},
		},
	}

	s.budgerigar.PutMany(stub)

	// Verify stub was added
	stubs := s.budgerigar.All()
	s.Require().Len(stubs, 1)

	// Purge stubs
	w := httptest.NewRecorder()
	s.server.PurgeStubs(w, nil)

	s.Equal(http.StatusNoContent, w.Code)

	// Verify stubs were purged
	stubs = s.budgerigar.All()
	s.Empty(stubs)
}

// TestSearchStubs tests stub search functionality.
func (s *RestServerTestSuite) TestSearchStubs() {
	// Add a stub
	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Input: stuber.InputData{
			Contains: map[string]any{"key": "value"},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": "success"},
		},
	}

	s.budgerigar.PutMany(stub)

	// Search stubs
	searchRequest := map[string]any{
		"service": "test.Service",
		"method":  "TestMethod",
		"data":    map[string]any{"key": "value"},
	}

	jsonData, err := json.Marshal(searchRequest)
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/stubs/search", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.server.SearchStubs(w, req)

	s.Equal(http.StatusOK, w.Code)

	// SearchStubs returns Output, not stub
	var response map[string]any

	err = json.Unmarshal(w.Body.Bytes(), &response)

	s.Require().NoError(err)
	s.Contains(response, "data")
}

// TestServiceMethodsList tests listing service methods.
func (s *RestServerTestSuite) TestServiceMethodsList() {
	w := httptest.NewRecorder()
	s.server.ServiceMethodsList(w, httptest.NewRequest(http.MethodGet, "/services/test.Service/methods", nil), "test.Service")

	s.Equal(http.StatusOK, w.Code)
}

// TestServicesList tests listing all services.
func (s *RestServerTestSuite) TestServicesList() {
	w := httptest.NewRecorder()
	s.server.ServicesList(w, nil)

	s.Equal(http.StatusOK, w.Code)

	// Just check that response is not empty and contains valid JSON
	s.NotEmpty(w.Body.String())
}

// TestValidateStubIntegration tests stub validation integration.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *RestServerTestSuite) TestValidateStubIntegration() {
	tests := []struct {
		name           string
		jsonData       string
		expectedStatus int
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
				"method": "TestBidirectional",
				"inputs": [{"contains": {"key": "value"}}],
				"output": {"stream": [{"result": "response"}]}
			}]`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing service",
			jsonData: `[{
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}},
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing method",
			jsonData: `[{
				"service": "test.Service",
				"input": {"contains": {"key": "value"}},
				"output": {"data": {"result": "success"}}
			}]`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			s.server.AddStub(w, req)

			s.Equal(tt.expectedStatus, w.Code)
		})
	}
}

// TestAddStubWithDelay tests stub addition with delay functionality via REST API.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *RestServerTestSuite) TestAddStubWithDelay() {
	tests := []struct {
		name           string
		jsonData       string
		expectedStatus int
		description    string
	}{
		{
			name: "unary stub with string delay",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}},
				"output": {
					"data": {"result": "success"},
					"delay": "100ms"
				}
			}]`,
			expectedStatus: http.StatusOK,
			description:    "should accept delay in string format (100ms)",
		},
		{
			name: "unary stub with longer delay",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}},
				"output": {
					"data": {"result": "success"},
					"delay": "2s"
				}
			}]`,
			expectedStatus: http.StatusOK,
			description:    "should accept delay in string format (2s)",
		},
		{
			name: "client stream stub with delay",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestClientStream",
				"inputs": [{"contains": {"key": "value"}}],
				"output": {
					"data": {"result": "success"},
					"delay": "500ms"
				}
			}]`,
			expectedStatus: http.StatusOK,
			description:    "should accept delay in client streaming stub",
		},
		{
			name: "server stream stub with delay",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestServerStream",
				"input": {"contains": {"key": "value"}},
				"output": {
					"stream": [{"result": "response"}],
					"delay": "1s"
				}
			}]`,
			expectedStatus: http.StatusOK,
			description:    "should accept delay in server streaming stub",
		},
		{
			name: "bidirectional stub with delay",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestBidirectional",
				"inputs": [{"contains": {"key": "value"}}],
				"output": {
					"stream": [{"result": "response"}],
					"delay": "750ms"
				}
			}]`,
			expectedStatus: http.StatusOK,
			description:    "should accept delay in bidirectional streaming stub",
		},
		{
			name: "unary stub without delay",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}},
				"output": {
					"data": {"result": "success"}
				}
			}]`,
			expectedStatus: http.StatusOK,
			description:    "should work without delay field",
		},
		{
			name: "unary stub with zero delay",
			jsonData: `[{
				"service": "test.Service",
				"method": "TestMethod",
				"input": {"contains": {"key": "value"}},
				"output": {
					"data": {"result": "success"},
					"delay": "0s"
				}
			}]`,
			expectedStatus: http.StatusOK,
			description:    "should work with zero delay",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Clear storage before each test
			s.budgerigar.Clear()

			req := httptest.NewRequest(http.MethodPost, "/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			s.server.AddStub(w, req)

			s.Equal(tt.expectedStatus, w.Code, tt.description)

			if tt.expectedStatus == http.StatusOK {
				// Verify that stub was added successfully
				var response []string

				err := json.Unmarshal(w.Body.Bytes(), &response)
				s.Require().NoError(err, "should unmarshal response as array of UUIDs")
				s.Len(response, 1, "should return exactly one UUID")

				// Verify that the stub exists in storage
				stubs := s.budgerigar.All()
				s.Len(stubs, 1, "should have exactly one stub in storage")

				// Verify that stub was added correctly
				stub := stubs[0]
				s.Equal("test.Service", stub.Service)

				// Check delay based on test case
				if tt.name == "unary stub without delay" || tt.name == "unary stub with zero delay" {
					s.Zero(stub.Output.Delay, "delay should be zero for this test case")
				} else {
					s.NotZero(stub.Output.Delay, "delay should be set for this test case")
				}
			}
		})
	}
}

// TestRestServerTestSuite runs the test suite.
func TestRestServerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RestServerTestSuite))
}
