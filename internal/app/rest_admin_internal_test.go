package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// AdminPanelTestSuite provides test suite for admin panel clever cases.
type AdminPanelTestSuite struct {
	suite.Suite

	server     *RestServer
	budgerigar *stuber.Budgerigar
}

// SetupSuite initializes the test suite.
func (s *AdminPanelTestSuite) SetupSuite() {
	s.budgerigar = stuber.NewBudgerigar(features.New())
	server, err := NewRestServer(context.Background(), s.budgerigar, nil)
	s.Require().NoError(err)
	s.server = server
}

// SetupTest cleans up before each test.
func (s *AdminPanelTestSuite) SetupTest() {
	s.budgerigar.Clear()
}

// TestSearchStubsWithRequestInternalHeader tests that internal requests don't mark stubs as used.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *AdminPanelTestSuite) TestSearchStubsWithRequestInternalHeader() {
	// Add a test stub
	stubData := `[{
		"service": "TestService",
		"method": "TestMethod",
		"input": {"equals": {"key": "test_value"}},
		"output": {"data": {"result": "internal_search_test"}}
	}]`

	// Add stub
	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	// Verify stub was added and is unused initially
	unusedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err := json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Require().Len(unusedStubs, 1)

	// Search for the stub WITH X-Gripmock-Requestinternal header (internal request)
	searchData := `{
		"service": "TestService",
		"method": "TestMethod",
		"data": {"key": "test_value"}
	}`

	searchReq := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
	searchReq.Header.Set("Content-Type", "application/json")
	searchReq.Header.Set("X-Gripmock-Requestinternal", "true") // This is the clever part!

	searchW := httptest.NewRecorder()

	s.server.SearchStubs(searchW, searchReq)
	s.Equal(http.StatusOK, searchW.Code)

	// Verify the stub was found
	var searchResult map[string]any

	err = json.Unmarshal(searchW.Body.Bytes(), &searchResult)
	s.Require().NoError(err)

	// SearchStubs returns stuber.Output structure
	data, ok := searchResult["data"].(map[string]any)
	s.Require().True(ok, "Response should have data field")
	s.Equal("internal_search_test", data["result"])

	// NOW THE CLEVER PART: Check that this stub is still in unused stubs
	// because the search with X-Gripmock-Requestinternal should not mark it as used
	unusedReq2 := httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
	unusedW2 := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW2, unusedReq2)
	s.Require().Equal(http.StatusOK, unusedW2.Code)

	var unusedStubsAfter []*stuber.Stub

	err = json.Unmarshal(unusedW2.Body.Bytes(), &unusedStubsAfter)
	s.Require().NoError(err)
	s.Require().Len(unusedStubsAfter, 1, "Stub should still be unused after internal search")

	// And check that it's NOT in used stubs
	usedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err = json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Require().Empty(usedStubs, "No stubs should be marked as used after internal search")
}

// TestSearchStubsWithoutRequestInternalHeader tests normal stub usage tracking.
func (s *AdminPanelTestSuite) TestSearchStubsWithoutRequestInternalHeader() {
	// Add a test stub
	stubData := `[{
		"service": "TestService",
		"method": "TestMethod",
		"input": {"equals": {"key": "normal_test"}},
		"output": {"data": {"result": "normal_search_test"}}
	}]`

	// Add stub
	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	// Search for the stub WITHOUT X-Gripmock-Requestinternal header (normal request)
	searchData := `{
		"service": "TestService",
		"method": "TestMethod",
		"data": {"key": "normal_test"}
	}`

	searchReq := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
	searchReq.Header.Set("Content-Type", "application/json")
	// NOTE: No X-Gripmock-Requestinternal header
	searchW := httptest.NewRecorder()

	s.server.SearchStubs(searchW, searchReq)
	s.Equal(http.StatusOK, searchW.Code)

	// Check that the stub is now in used stubs
	usedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err := json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Require().Len(usedStubs, 1, "Stub should be marked as used after normal search")

	// And check that it's NOT in unused stubs anymore
	unusedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err = json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Require().Empty(unusedStubs, "No stubs should be unused after normal search")
}

// TestMultipleInternalSearches tests that multiple internal searches don't affect usage stats.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *AdminPanelTestSuite) TestMultipleInternalSearches() {
	// Add multiple test stubs
	stubData := `[
		{
			"service": "TestService",
			"method": "Method1",
			"input": {"equals": {"id": 1}},
			"output": {"data": {"result": "method1_result"}}
		},
		{
			"service": "TestService",
			"method": "Method2", 
			"input": {"equals": {"id": 2}},
			"output": {"data": {"result": "method2_result"}}
		}
	]`

	// Add stubs
	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	// Search for both stubs multiple times with internal header
	searches := []string{
		`{"service": "TestService", "method": "Method1", "data": {"id": 1}}`,
		`{"service": "TestService", "method": "Method2", "data": {"id": 2}}`,
		`{"service": "TestService", "method": "Method1", "data": {"id": 1}}`, // Repeat
	}

	for i, searchData := range searches {
		searchReq := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
		searchReq.Header.Set("Content-Type", "application/json")
		searchReq.Header.Set("X-Gripmock-Requestinternal", "true")

		searchW := httptest.NewRecorder()

		s.server.SearchStubs(searchW, searchReq)
		s.Equal(http.StatusOK, searchW.Code, "Search %d should succeed", i+1)
	}

	// Verify all stubs are still unused
	unusedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err := json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Len(unusedStubs, 2, "All stubs should still be unused after multiple internal searches")

	// Verify no stubs are marked as used
	usedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err = json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Require().Empty(usedStubs, "No stubs should be used after internal searches")
}

// TestMixedInternalAndNormalSearches tests mixing internal and normal searches.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *AdminPanelTestSuite) TestMixedInternalAndNormalSearches() {
	// Add test stubs
	stubData := `[
		{
			"service": "TestService",
			"method": "InternalMethod",
			"input": {"equals": {"type": "internal"}},
			"output": {"data": {"result": "internal_only"}}
		},
		{
			"service": "TestService",
			"method": "NormalMethod",
			"input": {"equals": {"type": "normal"}},
			"output": {"data": {"result": "normal_only"}}
		}
	]`

	// Add stubs
	addReq := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	// Search first stub with internal header
	searchData1 := `{"service": "TestService", "method": "InternalMethod", "data": {"type": "internal"}}`
	searchReq1 := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData1))
	searchReq1.Header.Set("Content-Type", "application/json")
	searchReq1.Header.Set("X-Gripmock-Requestinternal", "true")

	searchW1 := httptest.NewRecorder()

	s.server.SearchStubs(searchW1, searchReq1)
	s.Equal(http.StatusOK, searchW1.Code)

	// Search second stub WITHOUT internal header
	searchData2 := `{"service": "TestService", "method": "NormalMethod", "data": {"type": "normal"}}`
	searchReq2 := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData2))
	searchReq2.Header.Set("Content-Type", "application/json")
	// No X-Gripmock-Requestinternal header
	searchW2 := httptest.NewRecorder()

	s.server.SearchStubs(searchW2, searchReq2)
	s.Equal(http.StatusOK, searchW2.Code)

	// Check unused stubs - should contain only the "internal" one
	unusedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err := json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Len(unusedStubs, 1, "Only internal search stub should be unused")
	s.Equal("InternalMethod", unusedStubs[0].Method)

	// Check used stubs - should contain only the "normal" one
	usedReq := httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err = json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Len(usedStubs, 1, "Only normal search stub should be used")
	s.Equal("NormalMethod", usedStubs[0].Method)
}

// TestAdminPanelTestSuite runs the admin panel test suite.
func TestAdminPanelTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdminPanelTestSuite))
}
