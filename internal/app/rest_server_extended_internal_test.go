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
	"github.com/bavix/gripmock/v3/internal/domain/rest"
)

// RestServerExtendedTestSuite provides extended test suite for REST server functionality.
type RestServerExtendedTestSuite struct {
	suite.Suite

	server     *RestServer
	budgerigar *stuber.Budgerigar
}

// SetupSuite initializes the test suite.
func (s *RestServerExtendedTestSuite) SetupSuite() {
	s.budgerigar = stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(context.Background(), s.budgerigar, extender)
	s.Require().NoError(err)
	s.server = server
}

// SetupTest cleans up before each test.
func (s *RestServerExtendedTestSuite) SetupTest() {
	s.budgerigar.Clear()
}

// TestAddStubWithPriority tests adding stubs with different priorities.
func (s *RestServerExtendedTestSuite) TestAddStubWithPriority() {
	tests := []struct {
		name     string
		jsonData string
		priority uint32
	}{
		{
			name: "high_priority_stub",
			jsonData: `[{
				"service": "PriorityService",
				"method": "HighPriority",
				"input": {"equals": {"key": "high"}},
				"output": {"data": {"priority": "high"}},
				"priority": 10
			}]`,
			priority: 10,
		},
		{
			name: "low_priority_stub",
			jsonData: `[{
				"service": "PriorityService", 
				"method": "LowPriority",
				"input": {"equals": {"key": "low"}},
				"output": {"data": {"priority": "low"}},
				"priority": 1
			}]`,
			priority: 1,
		},
		{
			name: "default_priority_stub",
			jsonData: `[{
				"service": "PriorityService",
				"method": "DefaultPriority", 
				"input": {"equals": {"key": "default"}},
				"output": {"data": {"priority": "default"}}
			}]`,
			priority: 0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			s.server.AddStub(w, req)

			s.Require().Equal(http.StatusOK, w.Code)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestAddStubWithHeaders tests adding stubs with header matching.
func (s *RestServerExtendedTestSuite) TestAddStubWithHeaders() {
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name: "stub_with_request_headers",
			jsonData: `[{
				"service": "HeaderService",
				"method": "WithHeaders",
				"input": {
					"equals": {"key": "value"},
					"headers": {"Authorization": "Bearer token", "Content-Type": "application/json"}
				},
				"output": {"data": {"authenticated": true}}
			}]`,
		},
		{
			name: "stub_with_response_headers",
			jsonData: `[{
				"service": "HeaderService",
				"method": "ResponseHeaders",
				"input": {"equals": {"key": "value"}},
				"output": {
					"data": {"result": "success"},
					"headers": {"X-Response-ID": "12345", "Cache-Control": "no-cache"}
				}
			}]`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			s.server.AddStub(w, req)

			s.Require().Equal(http.StatusOK, w.Code)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestAddStubWithMatchers tests different matcher types.
func (s *RestServerExtendedTestSuite) TestAddStubWithMatchers() {
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name: "contains_matcher",
			jsonData: `[{
				"service": "MatcherService",
				"method": "ContainsMatch",
				"input": {"contains": {"nested": {"field": "value"}}},
				"output": {"data": {"matched": "contains"}}
			}]`,
		},
		{
			name: "equals_matcher",
			jsonData: `[{
				"service": "MatcherService",
				"method": "EqualsMatch", 
				"input": {"equals": {"exact": "match"}},
				"output": {"data": {"matched": "equals"}}
			}]`,
		},
		{
			name: "matches_matcher",
			jsonData: `[{
				"service": "MatcherService",
				"method": "RegexMatch",
				"input": {"matches": {"pattern": "user_[0-9]+"}},
				"output": {"data": {"matched": "regex"}}
			}]`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			s.server.AddStub(w, req)

			s.Require().Equal(http.StatusOK, w.Code)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestAddStubWithErrors tests adding stubs that return errors.
func (s *RestServerExtendedTestSuite) TestAddStubWithErrors() {
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name: "stub_with_grpc_error",
			jsonData: `[{
				"service": "ErrorService",
				"method": "GrpcError",
				"input": {"equals": {"trigger": "error"}},
				"output": {"error": "Internal server error", "code": 13}
			}]`,
		},
		{
			name: "stub_with_custom_error",
			jsonData: `[{
				"service": "ErrorService", 
				"method": "CustomError",
				"input": {"equals": {"trigger": "custom"}},
				"output": {"error": "Custom error message", "code": 3}
			}]`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			s.server.AddStub(w, req)

			s.Require().Equal(http.StatusOK, w.Code)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestFindStubByIDExtended tests finding stubs by ID with various scenarios.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *RestServerExtendedTestSuite) TestFindStubByIDExtended() {
	// First add a stub to find
	stubData := `[{
		"service": "FindService",
		"method": "FindMethod", 
		"input": {"equals": {"key": "findme"}},
		"output": {"data": {"found": true}}
	}]`

	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()
	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	// Get stub ID by listing all stubs (since AddStub might return a message)
	listReq := httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
	listW := httptest.NewRecorder()
	s.server.ListStubs(listW, listReq)
	s.Require().Equal(http.StatusOK, listW.Code)

	var allStubs []*stuber.Stub

	err := json.Unmarshal(listW.Body.Bytes(), &allStubs)
	s.Require().NoError(err)
	s.Require().NotEmpty(allStubs)

	// Find our added stub
	var stubID uuid.UUID

	for _, stub := range allStubs {
		if stub.Service == "FindService" && stub.Method == "FindMethod" {
			stubID = stub.ID

			break
		}
	}

	s.Require().NotEqual(uuid.Nil, stubID, "Should find added stub")

	tests := []struct {
		name           string
		stubID         rest.ID
		expectedStatus int
		description    string
	}{
		{
			name:           "find_existing_stub",
			stubID:         stubID,
			expectedStatus: http.StatusOK,
			description:    "Should find existing stub",
		},
		{
			name:           "find_non_existing_stub",
			stubID:         uuid.New(),
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 for non-existing stub",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/api/stubs/"+tt.stubID.String(), nil)
			w := httptest.NewRecorder()

			s.server.FindByID(w, req, tt.stubID)

			s.Require().Equal(tt.expectedStatus, w.Code, tt.description)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestStubStatistics tests stub usage statistics endpoints.
func (s *RestServerExtendedTestSuite) TestStubStatistics() {
	// Add test stubs
	stubData := `[{
		"service": "StatsService",
		"method": "StatsMethod",
		"input": {"equals": {"trigger": "stats"}},
		"output": {"data": {"stats": true}}
	}]`

	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()
	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	tests := []struct {
		name        string
		endpoint    string
		description string
	}{
		{
			name:        "list_unused_stubs",
			endpoint:    "/api/stubs/unused",
			description: "Should list unused stubs",
		},
		{
			name:        "list_used_stubs",
			endpoint:    "/api/stubs/used",
			description: "Should list used stubs",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			switch tt.endpoint {
			case "/api/stubs/unused":
				s.server.ListUnusedStubs(w, req)
			case "/api/stubs/used":
				s.server.ListUsedStubs(w, req)
			}

			s.Require().Equal(http.StatusOK, w.Code, tt.description)
			s.Require().NotEmpty(w.Body.String())

			var stubs []*stuber.Stub

			err := json.Unmarshal(w.Body.Bytes(), &stubs)
			s.Require().NoError(err)
		})
	}
}

// TestSearchStubsExtended tests advanced stub searching.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *RestServerExtendedTestSuite) TestSearchStubsExtended() {
	// Add searchable stubs
	stubData := `[
		{
			"service": "SearchService",
			"method": "SearchMethod",
			"input": {"equals": {"search": "findme"}},
			"output": {"data": {"result": "found"}}
		},
		{
			"service": "SearchService", 
			"method": "AnotherMethod",
			"input": {"contains": {"partial": "match"}},
			"output": {"data": {"result": "partial"}}
		}
	]`

	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()
	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	tests := []struct {
		name           string
		searchData     string
		expectedStatus int
		description    string
	}{
		{
			name:           "search_exact_match",
			searchData:     `{"service": "SearchService", "method": "SearchMethod", "data": {"search": "findme"}}`,
			expectedStatus: http.StatusOK,
			description:    "Should find exact match",
		},
		{
			name:           "search_partial_match",
			searchData:     `{"service": "SearchService", "method": "AnotherMethod", "data": {"partial": "match"}}`,
			expectedStatus: http.StatusOK,
			description:    "Should find partial match",
		},
		{
			name:           "search_no_match",
			searchData:     `{"service": "NonExistent", "method": "NoMethod", "data": {"key": "novalue"}}`,
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 for no match",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(tt.searchData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			s.server.SearchStubs(w, req)

			s.Require().Equal(tt.expectedStatus, w.Code, tt.description)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestServiceDiscovery tests service and method discovery endpoints.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *RestServerExtendedTestSuite) TestServiceDiscovery() {
	// Add stubs for different services
	stubData := `[
		{
			"service": "com.example.UserService",
			"method": "GetUser",
			"input": {"equals": {"userId": "123"}},
			"output": {"data": {"user": "john"}}
		},
		{
			"service": "com.example.UserService",
			"method": "UpdateUser", 
			"input": {"equals": {"userId": "123"}},
			"output": {"data": {"updated": true}}
		},
		{
			"service": "com.example.OrderService",
			"method": "CreateOrder",
			"input": {"equals": {"productId": "456"}},
			"output": {"data": {"orderId": "789"}}
		}
	]`

	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()
	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	tests := []struct {
		name        string
		endpoint    string
		serviceName string
		description string
	}{
		{
			name:        "list_all_services",
			endpoint:    "/api/services",
			description: "Should list all services",
		},
		{
			name:        "list_user_service_methods",
			endpoint:    "/api/services/com.example.UserService/methods",
			serviceName: "com.example.UserService",
			description: "Should list UserService methods",
		},
		{
			name:        "list_order_service_methods",
			endpoint:    "/api/services/com.example.OrderService/methods",
			serviceName: "com.example.OrderService",
			description: "Should list OrderService methods",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			if tt.serviceName != "" {
				s.server.ServiceMethodsList(w, req, tt.serviceName)
			} else {
				s.server.ServicesList(w, req)
			}

			s.Require().Equal(http.StatusOK, w.Code, tt.description)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestHealthEndpoints tests health check endpoints.
func (s *RestServerExtendedTestSuite) TestHealthEndpoints() {
	tests := []struct {
		name        string
		endpoint    string
		handler     func(w http.ResponseWriter, r *http.Request)
		description string
	}{
		{
			name:        "liveness_check",
			endpoint:    "/api/health/liveness",
			handler:     s.server.Liveness,
			description: "Should return liveness status",
		},
		{
			name:        "readiness_check",
			endpoint:    "/api/health/readiness",
			handler:     s.server.Readiness,
			description: "Should return readiness status",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			// Health endpoints should return valid status codes
			s.Require().True(
				w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable,
				tt.description,
			)
			s.Require().NotEmpty(w.Body.String())
		})
	}
}

// TestBatchOperations tests batch operations on stubs.
func (s *RestServerExtendedTestSuite) TestBatchOperations() {
	// Add multiple stubs
	stubData := `[
		{
			"service": "BatchService",
			"method": "Method1",
			"input": {"equals": {"key": "value1"}},
			"output": {"data": {"result": "batch1"}}
		},
		{
			"service": "BatchService",
			"method": "Method2", 
			"input": {"equals": {"key": "value2"}},
			"output": {"data": {"result": "batch2"}}
		},
		{
			"service": "BatchService",
			"method": "Method3",
			"input": {"equals": {"key": "value3"}},
			"output": {"data": {"result": "batch3"}}
		}
	]`

	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()
	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	// Get added stub IDs by listing all stubs
	listReq := httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
	listW := httptest.NewRecorder()
	s.server.ListStubs(listW, listReq)
	s.Require().Equal(http.StatusOK, listW.Code)

	var allStubs []*stuber.Stub

	err := json.Unmarshal(listW.Body.Bytes(), &allStubs)
	s.Require().NoError(err)
	s.Require().Len(allStubs, 3)

	// Test batch deletion
	stubIDs := make([]string, len(allStubs))
	for i, stub := range allStubs {
		stubIDs[i] = stub.ID.String()
	}

	batchDeleteData, err := json.Marshal(stubIDs)
	s.Require().NoError(err)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/stubs", bytes.NewBuffer(batchDeleteData))
	deleteReq.Header.Set("Content-Type", "application/json")

	deleteW := httptest.NewRecorder()

	s.server.BatchStubsDelete(deleteW, deleteReq)

	s.Require().Equal(http.StatusOK, deleteW.Code)
}

// TestStubPersistence tests stub persistence across operations.
func (s *RestServerExtendedTestSuite) TestStubPersistence() {
	// Add a stub
	stubData := `[{
		"service": "PersistenceService",
		"method": "PersistentMethod",
		"input": {"equals": {"persistent": "data"}},
		"output": {"data": {"persisted": true}}
	}]`

	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()
	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	// Verify stub exists in listing
	listReq := httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
	listW := httptest.NewRecorder()
	s.server.ListStubs(listW, listReq)
	s.Require().Equal(http.StatusOK, listW.Code)

	var listedStubs []*stuber.Stub

	err := json.Unmarshal(listW.Body.Bytes(), &listedStubs)
	s.Require().NoError(err)
	s.Require().NotEmpty(listedStubs)

	// Find our stub
	var foundStub *stuber.Stub

	for _, stub := range listedStubs {
		if stub.Service == "PersistenceService" && stub.Method == "PersistentMethod" {
			foundStub = stub

			break
		}
	}

	s.Require().NotNil(foundStub, "Added stub should be found in listing")

	// Search for the stub
	searchData := `{"service": "PersistenceService", "method": "PersistentMethod", "data": {"persistent": "data"}}`
	searchReq := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
	searchReq.Header.Set("Content-Type", "application/json")

	searchW := httptest.NewRecorder()
	s.server.SearchStubs(searchW, searchReq)
	s.Require().Equal(http.StatusOK, searchW.Code)
}

// TestRestServerExtendedTestSuite runs the extended REST server test suite.
func TestRestServerExtendedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RestServerExtendedTestSuite))
}
