package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
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
	server, err := NewRestServer(s.T().Context(), s.budgerigar, extender, nil, nil, nil)
	s.Require().NoError(err)
	s.server = server
}

// SetupTest cleans up before each test.
func (s *RestServerTestSuite) SetupTest() {
	s.budgerigar.Clear()
}

// TestNewRestServer tests REST server creation.
func (s *RestServerTestSuite) TestNewRestServer() {
	ctx := s.T().Context()
	budgerigar := stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}

	server, err := NewRestServer(ctx, budgerigar, extender, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NotNil(server)
}

// TestAddStub tests stub addition functionality.
//
//nolint:funlen
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

	w := httptest.NewRecorder()
	s.server.DeleteStubByID(w, httptest.NewRequest(http.MethodDelete, "/", nil), stubID)

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
	s.server.ListStubs(w, httptest.NewRequest(http.MethodGet, "/", nil))

	s.Equal(http.StatusOK, w.Code)

	// ListStubs returns array of stubs
	var response []*stuber.Stub

	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	s.Len(response, 1)
}

// TestAddDescriptors tests POST /api/descriptors with binary FileDescriptorSet.
func (s *RestServerTestSuite) TestAddDescriptors() {
	body := s.greeterDescriptorSetBytes()
	w := s.addDescriptorsPayload(s.server, body)
	s.Equal(http.StatusOK, w.Code)

	var addResp struct {
		Message    string   `json:"message"`
		ServiceIDs []string `json:"serviceIDs"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &addResp)
	s.Require().NoError(err)
	s.Equal("ok", addResp.Message)
	s.Contains(addResp.ServiceIDs, "helloworld.Greeter")

	// Verify service appears in ServicesList
	w2 := httptest.NewRecorder()
	s.server.ServicesList(w2, httptest.NewRequest(http.MethodGet, "/api/services", nil))
	s.Equal(http.StatusOK, w2.Code)

	var services []map[string]any
	s.Require().NoError(json.Unmarshal(w2.Body.Bytes(), &services))
	s.Require().NotEmpty(services)

	ids := make([]string, 0, len(services))
	for _, svc := range services {
		if id, ok := svc["id"].(string); ok {
			ids = append(ids, id)
		}
	}

	s.Contains(ids, "helloworld.Greeter")
}

// TestListDescriptors tests GET /api/descriptors.
func (s *RestServerTestSuite) TestListDescriptors() {
	// Add a descriptor first
	w := s.addDescriptorsPayload(s.server, s.greeterDescriptorSetBytes())
	s.Require().Equal(http.StatusOK, w.Code)

	// List descriptors - verify serviceID appears
	w2 := httptest.NewRecorder()
	s.server.ListDescriptors(w2, httptest.NewRequest(http.MethodGet, "/api/descriptors", nil))
	s.Equal(http.StatusOK, w2.Code)

	var resp struct {
		ServiceIDs []string `json:"serviceIDs"`
	}

	s.Require().NoError(json.Unmarshal(w2.Body.Bytes(), &resp))
	s.Require().NotEmpty(resp.ServiceIDs)
	s.Contains(resp.ServiceIDs, "helloworld.Greeter")
}

// TestDeleteService tests DELETE /api/services/{serviceID}.
func (s *RestServerTestSuite) TestDeleteService() {
	// Add descriptors
	w := s.addDescriptorsPayload(s.server, s.greeterDescriptorSetBytes())
	s.Require().Equal(http.StatusOK, w.Code)

	// Delete service
	delReq := httptest.NewRequest(http.MethodDelete, "/api/services/helloworld.Greeter", nil)
	delReq = mux.SetURLVars(delReq, map[string]string{"serviceID": "helloworld.Greeter"})

	w2 := httptest.NewRecorder()
	s.server.DeleteService(w2, delReq, "helloworld.Greeter")
	s.Equal(http.StatusNoContent, w2.Code)

	// Delete non-existent returns 404
	w3 := httptest.NewRecorder()
	s.server.DeleteService(w3, delReq, "helloworld.Greeter")
	s.Equal(http.StatusNotFound, w3.Code)
}

func (s *RestServerTestSuite) TestMcpInfo() {
	w := httptest.NewRecorder()
	s.server.McpInfo(w, httptest.NewRequest(http.MethodGet, "/api/mcp", nil))

	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("gripmock", resp["serverName"])
	s.Equal("2024-11-05", resp["protocolVersion"])
}

func (s *RestServerTestSuite) TestMcpMessageDescriptorsLifecycle() {
	ctx := s.T().Context()
	protoPath := filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto")
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath})
	s.Require().NoError(err)
	body, err := proto.Marshal(fdsSlice[0])
	s.Require().NoError(err)

	initResp := s.mcpCallOK(s.server, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize"})
	s.Contains(initResp, "result")

	addResp := s.mcpToolCall(s.server, 2, "descriptors.add", map[string]any{
		"descriptorSetBase64": base64.StdEncoding.EncodeToString(body),
	})
	s.NotContains(addResp, "error")

	listResp := s.mcpToolCall(s.server, 3, "descriptors.list", map[string]any{})

	structured := s.mcpStructuredContent(listResp)
	_, ok := structured["serviceIDs"]
	s.True(ok)
}

func (s *RestServerTestSuite) TestAddStub_SessionFromHeader() {
	// Arrange
	stubJSON := `[
		{
			"service": "svc",
			"method": "M",
			"input": {"contains": {"x": "1"}},
			"output": {"data": {"ok": true}}
		}
	]`

	// Act
	w := s.addStubJSONWithRequest(s.server, stubJSON, func(req *http.Request) {
		req.Header.Set("X-Gripmock-Session", "header-session")
	})

	// Assert
	s.Equal(http.StatusOK, w.Code)

	stubs := s.budgerigar.All()
	s.Require().Len(stubs, 1)
	s.Equal("header-session", stubs[0].Session)
}

func (s *RestServerTestSuite) TestMcpMessageHistoryTools() {
	server := s.newRestServerWithHistory(
		history.CallRecord{Service: "svc", Method: "A", Request: map[string]any{"id": 1}},
		history.CallRecord{Service: "svc", Method: "B", Error: "boom"},
		history.CallRecord{Service: "svc", Method: "B", Error: "kaboom"},
	)

	listResp := s.mcpToolCall(server, 1, "history.list", map[string]any{
		"service": "svc",
		"method":  "B",
		"limit":   1,
	})

	listJSON := s.mcpStructuredContent(listResp)
	records, ok := listJSON["records"].([]any)
	s.Require().True(ok)
	s.Len(records, 1)

	errorResp := s.mcpToolCall(server, 2, "history.errors", map[string]any{"limit": 1})

	errorJSON := s.mcpStructuredContent(errorResp)
	errorRecords, ok := errorJSON["records"].([]any)
	s.Require().True(ok)
	s.Len(errorRecords, 1)
}

func (s *RestServerTestSuite) TestMcpDebugCallTool() {
	server := s.newRestServerWithHistory(history.CallRecord{Service: "svc.Greeter", Method: "SayHello", Error: "not matched"})

	stubData := `[
		{
			"service":"svc.Greeter",
			"method":"SayHello",
			"input":{"equals":{"name":"Alex"}},
			"output":{"data":{"message":"Hi"}}
		}
	]`

	s.addStubJSON(server, stubData)

	response := s.mcpToolCall(server, 1, "debug.call", map[string]any{
		"service": "svc.Greeter",
		"method":  "SayHello",
		"limit":   5,
	})

	jsonPayload := s.mcpStructuredContent(response)

	stubCount, ok := jsonPayload["stubCount"].(float64)
	s.Require().True(ok)
	s.InDelta(1.0, stubCount, 0.0)

	errorCount, ok := jsonPayload["errorCount"].(float64)
	s.Require().True(ok)
	s.InDelta(1.0, errorCount, 0.0)
}

func (s *RestServerTestSuite) TestMcpToolCallReturnsStructuredContent() {
	response := s.mcpToolCall(s.server, 1, "descriptors.list", map[string]any{})

	result, ok := response["result"].(map[string]any)
	s.Require().True(ok)

	content, ok := result["content"].([]any)
	s.Require().True(ok)
	s.Require().NotEmpty(content)

	contentItem, ok := content[0].(map[string]any)
	s.Require().True(ok)
	s.Equal("text", contentItem["type"])
	s.Equal("OK", contentItem["text"])

	structured, ok := result["structuredContent"].(map[string]any)
	s.Require().True(ok)

	_, hasServiceIDs := structured["serviceIDs"]
	s.True(hasServiceIDs)
}

func (s *RestServerTestSuite) TestMcpToolCallInvalidParamsCode() {
	response := s.mcpToolCall(s.server, 1, "history.list", map[string]any{"limit": -1})
	s.InDelta(float64(-32602), s.mcpErrorCode(response), 0.0)
}

func (s *RestServerTestSuite) TestMcpToolCallUnknownToolCode() {
	response := s.mcpToolCall(s.server, 1, "unknown.tool", map[string]any{})
	s.InDelta(float64(-32601), s.mcpErrorCode(response), 0.0)
}

func (s *RestServerTestSuite) TestMcpNotificationReturnsNoContent() {
	s.mcpCall(s.server, map[string]any{
		"jsonrpc": "2.0",
		"method":  "ping",
	}, http.StatusNoContent)
}

func (s *RestServerTestSuite) TestMcpHistoryList_SessionScoping() {
	tests := []struct {
		name           string
		arguments      map[string]any
		prepareRequest func(*http.Request)
		expectedLen    int
	}{
		{
			name:        "uses session from header",
			arguments:   map[string]any{"service": "svc", "method": "M"},
			expectedLen: 2,
			prepareRequest: func(req *http.Request) {
				req.Header.Set("X-Gripmock-Session", "A")
			},
		},
		{
			name:           "without session returns all records",
			arguments:      map[string]any{"service": "svc", "method": "M"},
			expectedLen:    3,
			prepareRequest: nil,
		},
		{
			name: "explicit session overrides transport session",
			arguments: map[string]any{
				"service": "svc",
				"method":  "M",
				"session": "B",
			},
			expectedLen: 2,
			prepareRequest: func(req *http.Request) {
				req.Header.Set("X-Gripmock-Session", "A")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Arrange
			server := s.newRestServerWithSessionHistory()

			// Act
			response := s.mcpToolCallWithRequest(server, 1, "history.list", tt.arguments, tt.prepareRequest)

			// Assert
			structured := s.mcpStructuredContent(response)
			records, ok := structured["records"].([]any)
			s.Require().True(ok)
			s.Len(records, tt.expectedLen)
		})
	}
}

func (s *RestServerTestSuite) TestMcpDebugCall_SessionScoping() {
	tests := []struct {
		name           string
		arguments      map[string]any
		prepareRequest func(*http.Request)
		expectedCount  float64
	}{
		{
			name:      "uses session from transport for stub scope",
			arguments: map[string]any{"service": "svc.Greeter", "method": "SayHello"},
			prepareRequest: func(req *http.Request) {
				req.Header.Set("X-Gripmock-Session", "A")
			},
			expectedCount: 2.0,
		},
		{
			name: "explicit session overrides transport session for stub scope",
			arguments: map[string]any{
				"service": "svc.Greeter",
				"method":  "SayHello",
				"session": "B",
			},
			prepareRequest: func(req *http.Request) {
				req.Header.Set("X-Gripmock-Session", "A")
			},
			expectedCount: 2.0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Arrange
			server := s.newRestServerWithHistory()
			s.addDebugScopeStubs(server)

			// Act
			response := s.mcpToolCallWithRequest(server, 1, "debug.call", tt.arguments, tt.prepareRequest)

			// Assert
			structured := s.mcpStructuredContent(response)
			stubCount, ok := structured["stubCount"].(float64)
			s.Require().True(ok)
			s.InDelta(tt.expectedCount, stubCount, 0.0)
		})
	}
}

func (s *RestServerTestSuite) TestListStubs_EmptyByUsage() {
	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{name: "unused stubs", handler: s.server.ListUnusedStubs},
		{name: "used stubs", handler: s.server.ListUsedStubs},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Arrange
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			// Act
			w := httptest.NewRecorder()
			tt.handler(w, req)

			// Assert
			s.Equal(http.StatusOK, w.Code)

			var response []*stuber.Stub

			err := json.Unmarshal(w.Body.Bytes(), &response)
			s.Require().NoError(err)
			s.Empty(response)
		})
	}
}

// TestListHistory tests history endpoint (empty when history is disabled).
func (s *RestServerTestSuite) TestListHistory() {
	w := httptest.NewRecorder()
	s.server.ListHistory(w, httptest.NewRequest(http.MethodGet, "/api/history", nil))
	s.Equal(http.StatusOK, w.Code)

	var list []any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &list))
	s.Empty(list)
}

func (s *RestServerTestSuite) TestListHistory_SessionFromHeader() {
	// Arrange
	server := s.newRestServerWithSessionHistory()

	// Act
	w := s.listHistory(server, func(req *http.Request) {
		req.Header.Set("X-Gripmock-Session", "A")
	})
	s.Equal(http.StatusOK, w.Code)

	// Assert
	var list []map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &list))
	s.Len(list, 2)
}

// TestListHistory_RedactsSensitiveKeys tests that ListHistory returns redacted values when store has redact keys.
func (s *RestServerTestSuite) TestListHistory_RedactsSensitiveKeys() {
	store := history.NewMemoryStore(0, history.WithRedactKeys([]string{"password", "token"}))
	store.Record(history.CallRecord{
		Service:  "svc",
		Method:   "M",
		Request:  map[string]any{"user": "alice", "password": "secret123"},
		Response: map[string]any{"token": "jwt-xxx"},
	})

	server := s.newRestServerWithStore(store)

	w := s.listHistory(server, nil)
	s.Equal(http.StatusOK, w.Code)

	var list []map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &list))
	s.Len(list, 1)
	req, ok := list[0]["request"].(map[string]any)
	s.Require().True(ok)
	resp, ok := list[0]["response"].(map[string]any)
	s.Require().True(ok)
	s.Equal("alice", req["user"])
	s.Equal("[REDACTED]", req["password"])
	s.Equal("[REDACTED]", resp["token"])
}

// TestVerifyCallsWithHistory tests verify endpoint with history store.
func (s *RestServerTestSuite) TestVerifyCallsWithHistory() {
	server := s.newRestServerWithHistory(
		history.CallRecord{Service: "svc", Method: "M", Request: map[string]any{"x": "1"}},
		history.CallRecord{Service: "svc", Method: "M", Request: map[string]any{"x": "2"}},
	)

	// GET /api/history returns records
	w := s.listHistory(server, nil)
	s.Equal(http.StatusOK, w.Code)

	var list []map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &list))
	s.Len(list, 2)

	// POST /api/verify with correct count passes
	w2 := s.verifyCalls(server, map[string]any{"service": "svc", "method": "M", "expectedCount": 2}, nil)
	s.Equal(http.StatusOK, w2.Code)

	// Wrong count fails
	w3 := s.verifyCalls(server, map[string]any{"service": "svc", "method": "M", "expectedCount": 1}, nil)
	s.Equal(http.StatusBadRequest, w3.Code)
}

func (s *RestServerTestSuite) TestVerifyCallsWithHistory_SessionFromHeader() {
	// Arrange
	server := s.newRestServerWithSessionHistory()

	// Act
	w := s.verifyCalls(server, map[string]any{"service": "svc", "method": "M", "expectedCount": 2}, func(req *http.Request) {
		req.Header.Set("X-Gripmock-Session", "A")
	})

	// Assert
	s.Equal(http.StatusOK, w.Code)
}

// TestLiveness tests liveness endpoint.
func (s *RestServerTestSuite) TestLiveness() {
	w := httptest.NewRecorder()
	s.server.Liveness(w, httptest.NewRequest(http.MethodGet, "/", nil))

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
			s.server.Readiness(w, httptest.NewRequest(http.MethodGet, "/", nil))

			if w.Code == http.StatusOK {
				// Server is ready, final check
				w := httptest.NewRecorder()
				s.server.Readiness(w, httptest.NewRequest(http.MethodGet, "/", nil))
				s.Equal(http.StatusOK, w.Code)

				return
			}
		}
	}
}

// TestPurgeStubs tests purging all stubs.
func (s *RestServerTestSuite) TestPurgeStubs() {
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
	s.server.PurgeStubs(w, httptest.NewRequest(http.MethodDelete, "/", nil))

	s.Equal(http.StatusNoContent, w.Code)

	// Verify stubs were purged
	stubs = s.budgerigar.All()
	s.Empty(stubs)
}

// TestSearchStubs tests stub search functionality.
func (s *RestServerTestSuite) TestSearchStubs() {
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

	w := s.searchStubs(s.server, searchRequest, nil)

	s.Equal(http.StatusOK, w.Code)

	// SearchStubs returns Output, not stub
	var response map[string]any

	err := json.Unmarshal(w.Body.Bytes(), &response)

	s.Require().NoError(err)
	s.Contains(response, "data")
}

func (s *RestServerTestSuite) TestSearchStubs_SessionFromHeader() {
	// Arrange
	s.budgerigar.PutMany(&stuber.Stub{
		Service: "svc",
		Method:  "M",
		Session: "A",
		Input: stuber.InputData{
			Contains: map[string]any{"x": "1"},
		},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	})

	searchBody, err := json.Marshal(map[string]any{
		"service": "svc",
		"method":  "M",
		"data":    map[string]any{"x": "1"},
	})
	s.Require().NoError(err)

	// Act
	wNoSession := s.searchStubs(s.server, nil, searchBody)
	s.Equal(http.StatusNotFound, wNoSession.Code)

	wWithSession := s.searchStubsWithRequest(s.server, nil, searchBody, func(req *http.Request) {
		req.Header.Set("X-Gripmock-Session", "A")
	})
	// Assert
	s.Equal(http.StatusOK, wWithSession.Code)
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
	s.server.ServicesList(w, httptest.NewRequest(http.MethodGet, "/", nil))

	s.Equal(http.StatusOK, w.Code)

	// Just check that response is not empty and contains valid JSON
	s.NotEmpty(w.Body.String())
}

// TestValidateStubIntegration tests stub validation integration.
//
//nolint:funlen
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
//nolint:funlen
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

func (s *RestServerTestSuite) mcpCall(server *RestServer, payload map[string]any, expectedStatus int) map[string]any {
	return s.mcpCallWithRequest(server, payload, expectedStatus, nil)
}

func (s *RestServerTestSuite) mcpCallWithRequest(
	server *RestServer,
	payload map[string]any,
	expectedStatus int,
	prepare func(*http.Request),
) map[string]any {
	requestBody, err := json.Marshal(payload)
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/api/mcp", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	if prepare != nil {
		prepare(req)
	}

	w := httptest.NewRecorder()
	server.McpMessage(w, req)
	s.Equal(expectedStatus, w.Code)

	if expectedStatus == http.StatusNoContent {
		s.Empty(w.Body.String())

		return nil
	}

	var response map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &response))

	return response
}

func (s *RestServerTestSuite) mcpCallOK(server *RestServer, payload map[string]any) map[string]any {
	return s.mcpCall(server, payload, http.StatusOK)
}

func (s *RestServerTestSuite) mcpToolCall(server *RestServer, id int, name string, arguments map[string]any) map[string]any {
	return s.mcpToolCallWithRequest(server, id, name, arguments, nil)
}

func (s *RestServerTestSuite) mcpToolCallWithRequest(
	server *RestServer,
	id int,
	name string,
	arguments map[string]any,
	prepare func(*http.Request),
) map[string]any {
	return s.mcpCallWithRequest(server, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	}, http.StatusOK, prepare)
}

func (s *RestServerTestSuite) newRestServerWithHistory(records ...history.CallRecord) *RestServer {
	store := history.NewMemoryStore(0)
	for _, record := range records {
		store.Record(record)
	}

	return s.newRestServerWithStore(store)
}

func (s *RestServerTestSuite) newRestServerWithStore(store history.Reader) *RestServer {
	server, err := NewRestServer(s.T().Context(), stuber.NewBudgerigar(features.New()), &mockExtender{}, store, nil, nil)
	s.Require().NoError(err)

	return server
}

func (s *RestServerTestSuite) newRestServerWithSessionHistory() *RestServer {
	return s.newRestServerWithHistory(
		history.CallRecord{Service: "svc", Method: "M", Session: ""},
		history.CallRecord{Service: "svc", Method: "M", Session: "A"},
		history.CallRecord{Service: "svc", Method: "M", Session: "B"},
	)
}

func (s *RestServerTestSuite) addStubJSON(server *RestServer, payload string) {
	w := s.addStubJSONWithRequest(server, payload, nil)
	s.Equal(http.StatusOK, w.Code)
}

func (s *RestServerTestSuite) greeterDescriptorSetBytes() []byte {
	ctx := s.T().Context()
	protoPath := filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto")
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath})
	s.Require().NoError(err)
	s.Require().NotEmpty(fdsSlice)

	body, err := proto.Marshal(fdsSlice[0])
	s.Require().NoError(err)

	return body
}

func (s *RestServerTestSuite) addDescriptorsPayload(server *RestServer, payload []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/descriptors", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/octet-stream")

	w := httptest.NewRecorder()
	server.AddDescriptors(w, req)

	return w
}

func (s *RestServerTestSuite) addStubJSONWithRequest(
	server *RestServer,
	payload string,
	prepare func(*http.Request),
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	if prepare != nil {
		prepare(req)
	}

	w := httptest.NewRecorder()
	server.AddStub(w, req)

	return w
}

func (s *RestServerTestSuite) addDebugScopeStubs(server *RestServer) {
	s.addStubJSONWithRequest(
		server,
		`[{"service":"svc.Greeter","method":"SayHello","input":{"equals":{"name":"Alex"}},"output":{"data":{"message":"Hi A"}}}]`,
		func(req *http.Request) {
			req.Header.Set("X-Gripmock-Session", "A")
		},
	)

	s.addStubJSONWithRequest(
		server,
		`[{"service":"svc.Greeter","method":"SayHello","input":{"equals":{"name":"Alex"}},"output":{"data":{"message":"Hi B"}}}]`,
		func(req *http.Request) {
			req.Header.Set("X-Gripmock-Session", "B")
		},
	)

	s.addStubJSON(server, `[
		{"service":"svc.Greeter","method":"SayHello","input":{"equals":{"name":"Alex"}},"output":{"data":{"message":"Hi Global"}}}
	]`)
}

func (s *RestServerTestSuite) listHistory(server *RestServer, prepare func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	if prepare != nil {
		prepare(req)
	}

	w := httptest.NewRecorder()
	server.ListHistory(w, req)

	return w
}

func (s *RestServerTestSuite) searchStubs(
	server *RestServer,
	payload map[string]any,
	raw []byte,
) *httptest.ResponseRecorder {
	return s.searchStubsWithRequest(server, payload, raw, nil)
}

func (s *RestServerTestSuite) searchStubsWithRequest(
	server *RestServer,
	payload map[string]any,
	raw []byte,
	prepare func(*http.Request),
) *httptest.ResponseRecorder {
	body := raw
	if body == nil {
		var err error

		body, err = json.Marshal(payload)
		s.Require().NoError(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/stubs/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if prepare != nil {
		prepare(req)
	}

	w := httptest.NewRecorder()
	server.SearchStubs(w, req)

	return w
}

func (s *RestServerTestSuite) verifyCalls(
	server *RestServer,
	payload map[string]any,
	prepare func(*http.Request),
) *httptest.ResponseRecorder {
	body, err := json.Marshal(payload)
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/api/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if prepare != nil {
		prepare(req)
	}

	w := httptest.NewRecorder()
	server.VerifyCalls(w, req)

	return w
}

func (s *RestServerTestSuite) mcpStructuredContent(response map[string]any) map[string]any {
	result, ok := response["result"].(map[string]any)
	s.Require().True(ok)

	structured, ok := result["structuredContent"].(map[string]any)
	s.Require().True(ok)

	return structured
}

func (s *RestServerTestSuite) mcpErrorCode(response map[string]any) float64 {
	errObj, ok := response["error"].(map[string]any)
	s.Require().True(ok)

	code, ok := errObj["code"].(float64)
	s.Require().True(ok)

	return code
}

// TestRestServerTestSuite runs the test suite.
func TestRestServerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RestServerTestSuite))
}
