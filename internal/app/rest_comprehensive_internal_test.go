package app

import (
	"bytes"
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

// mockExtender is imported from test_utils.go

// RestComprehensiveTestSuite provides comprehensive test suite for REST endpoints.
type RestComprehensiveTestSuite struct {
	suite.Suite

	server     *RestServer
	budgerigar *stuber.Budgerigar
}

// SetupSuite initializes the test suite.
func (s *RestComprehensiveTestSuite) SetupSuite() {
	s.budgerigar = stuber.NewBudgerigar(features.New())
	extender := &mockExtender{}
	server, err := NewRestServer(s.T().Context(), s.budgerigar, extender, nil, nil)
	s.Require().NoError(err)
	s.server = server
}

// SetupTest cleans up before each test.
func (s *RestComprehensiveTestSuite) SetupTest() {
	s.budgerigar.Clear()
}

// TestFindByID tests finding stubs by ID.
func (s *RestComprehensiveTestSuite) TestFindByID() {
	tests := []struct {
		name           string
		expectedStatus int
		description    string
	}{
		{
			name:           "find_non-existing_stub",
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 for non-existing stub",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Use a random UUID for non-existing stub test
			randomUUID := uuid.New()
			req := httptest.NewRequest(http.MethodGet, "/api/stubs/"+randomUUID.String(), nil)
			w := httptest.NewRecorder()

			s.server.FindByID(w, req, randomUUID)

			s.Equal(tt.expectedStatus, w.Code, tt.description)
			s.Require().NotEmpty(w.Body.String(), "Response should not be empty")
		})
	}
}

// TestBatchStubsDeleteComprehensive tests batch deletion functionality.
func (s *RestComprehensiveTestSuite) TestBatchStubsDeleteComprehensive() {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		description    string
	}{
		{
			name:           "delete_with_valid_UUIDs",
			requestBody:    `["550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"]`,
			expectedStatus: http.StatusOK,
			description:    "Should accept valid UUIDs",
		},
		{
			name:           "delete_with_empty_array",
			requestBody:    `[]`,
			expectedStatus: http.StatusOK,
			description:    "Should handle empty array",
		},
		{
			name:           "delete_with_invalid_JSON",
			requestBody:    `invalid json`,
			expectedStatus: http.StatusInternalServerError,
			description:    "Should reject invalid JSON",
		},
		{
			name:           "delete_with_invalid_UUIDs",
			requestBody:    `["invalid-uuid", "another-invalid"]`,
			expectedStatus: http.StatusInternalServerError,
			description:    "Should reject invalid UUIDs",
		},
		{
			name:           "delete_with_mixed_valid/invalid_UUIDs",
			requestBody:    `["550e8400-e29b-41d4-a716-446655440000", "invalid-uuid"]`,
			expectedStatus: http.StatusInternalServerError,
			description:    "Should reject mixed valid/invalid UUIDs",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodDelete, "/api/stubs", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			s.server.BatchStubsDelete(w, req)

			s.Equal(tt.expectedStatus, w.Code, tt.description)
		})
	}
}

// TestSearchStubsComprehensive tests search functionality.
func (s *RestComprehensiveTestSuite) TestSearchStubsComprehensive() {
	tests := []struct {
		name           string
		method         string
		requestBody    string
		expectedStatus int
		description    string
	}{
		{
			name:           "search_with_non-existing_service",
			method:         http.MethodPost,
			requestBody:    `{"service": "NonExistentService", "method": "TestMethod", "data": {}}`,
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 for non-existing service",
		},
		{
			name:           "search_without_query_params",
			method:         http.MethodPost,
			requestBody:    `{}`,
			expectedStatus: http.StatusNotFound,
			description:    "Should handle empty search query",
		},
		{
			name:           "search_with_empty_request_body",
			method:         http.MethodPost,
			requestBody:    ``,
			expectedStatus: http.StatusInternalServerError,
			description:    "Should handle empty request body",
		},
		{
			name:           "search_with_invalid_JSON_body",
			method:         http.MethodPost,
			requestBody:    `invalid json`,
			expectedStatus: http.StatusInternalServerError,
			description:    "Should reject invalid JSON",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(tt.method, "/api/stubs/search", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			s.server.SearchStubs(w, req)

			s.Equal(tt.expectedStatus, w.Code, tt.description)
		})
	}
}

// TestServiceMethodsListComprehensive tests service methods listing.
func (s *RestComprehensiveTestSuite) TestServiceMethodsListComprehensive() {
	tests := []struct {
		name        string
		serviceName string
		description string
	}{
		{
			name:        "get_methods_for_valid_service",
			serviceName: "test.Service",
			description: "Should handle valid service name",
		},
		{
			name:        "get_methods_for_non-existing_service",
			serviceName: "NonExistent.Service",
			description: "Should handle non-existing service",
		},
		{
			name:        "get_methods_for_service_with_empty_package",
			serviceName: "Service",
			description: "Should handle service without package",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/api/services/"+tt.serviceName+"/methods", nil)
			w := httptest.NewRecorder()

			s.server.ServiceMethodsList(w, req, tt.serviceName)

			// Should always return some response
			s.Require().NotEmpty(w.Body.String(), tt.description)
		})
	}
}

// TestReadinessComprehensive tests readiness endpoint.
func (s *RestComprehensiveTestSuite) TestReadinessComprehensive() {
	s.Run("readiness_check", func() {
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
				req := httptest.NewRequest(http.MethodGet, "/api/health/readiness", nil)
				w := httptest.NewRecorder()
				s.server.Readiness(w, req)

				if w.Code == http.StatusOK {
					// Server is ready, final check
					req := httptest.NewRequest(http.MethodGet, "/api/health/readiness", nil)
					w := httptest.NewRecorder()
					s.server.Readiness(w, req)
					s.Equal(http.StatusOK, w.Code)
					s.NotEmpty(w.Body.String(), "Response should not be empty")

					return
				}
			}
		}
	})
}

// TestErrorHandling tests general error responses.
func (s *RestComprehensiveTestSuite) TestErrorHandling() {
	tests := []struct {
		name        string
		method      string
		endpoint    string
		body        string
		description string
	}{
		{
			name:        "test_internal_server_error_handling",
			method:      http.MethodPost,
			endpoint:    "/stub",
			body:        `[{"invalid": "structure"}]`,
			description: "Should handle internal server errors gracefully",
		},
		{
			name:        "test_empty_body_handling",
			method:      http.MethodPost,
			endpoint:    "/stub",
			body:        "",
			description: "Should handle empty request body",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(tt.method, tt.endpoint, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			if tt.endpoint == "/stub" {
				s.server.AddStub(w, req)
			}

			// Should always return some response
			s.Require().NotEmpty(w.Body.String(), tt.description)

			// Check that error response has proper format
			if w.Code >= http.StatusBadRequest {
				var response map[string]any

				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err == nil {
					s.Contains([]string{"error", "message", "details"},
						getFirstKey(response), "Error response should have proper structure")
				}
			}
		})
	}
}

// TestStubLifecycle tests complete stub operations lifecycle.
func (s *RestComprehensiveTestSuite) TestStubLifecycle() {
	s.Run("complete_stub_lifecycle", func() {
		// 1. Add stub
		stubData := `[{
			"service": "TestService",
			"method": "TestMethod",
			"input": {"equals": {"key": "value"}},
			"output": {"data": {"result": "success"}}
		}]`

		addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
		addReq.Header.Set("Content-Type", "application/json")

		addW := httptest.NewRecorder()

		s.server.AddStub(addW, addReq)
		s.Equal(http.StatusOK, addW.Code, "Adding stub should succeed")
		s.NotEmpty(addW.Body.String(), "Add response should not be empty")

		// 2. List all stubs
		listReq := httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
		listW := httptest.NewRecorder()

		s.server.ListStubs(listW, listReq)
		s.Equal(http.StatusOK, listW.Code, "Listing stubs should succeed")
		s.NotEmpty(listW.Body.String(), "List response should not be empty")

		// 3. Search for stub
		searchData := `{"service": "TestService", "method": "TestMethod", "data": {"key": "value"}}`
		searchReq := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
		searchReq.Header.Set("Content-Type", "application/json")

		searchW := httptest.NewRecorder()

		s.server.SearchStubs(searchW, searchReq)
		s.Equal(http.StatusOK, searchW.Code, "Searching stub should succeed")
		s.NotEmpty(searchW.Body.String(), "Search response should not be empty")

		// 4. List used stubs
		usedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
		usedW := httptest.NewRecorder()

		s.server.ListUsedStubs(usedW, usedReq)
		s.Equal(http.StatusOK, usedW.Code, "Listing used stubs should succeed")
		s.NotEmpty(usedW.Body.String(), "Used stubs response should not be empty")

		// 5. Purge all stubs
		purgeReq := httptest.NewRequest(http.MethodDelete, "/api/stubs", nil)
		purgeW := httptest.NewRecorder()

		s.server.PurgeStubs(purgeW, purgeReq)
		s.Equal(http.StatusNoContent, purgeW.Code, "Purging stubs should succeed")
	})
}

// getFirstKey returns the first key from a map for error structure validation.
func getFirstKey(m map[string]any) string {
	for k := range m {
		return k
	}

	return ""
}

// TestRestComprehensiveTestSuite runs the comprehensive REST test suite.
func TestRestComprehensiveTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RestComprehensiveTestSuite))
}
