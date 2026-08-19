package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type AdminPanelTestSuite struct {
	suite.Suite

	server     *RestServer
	budgerigar *stuber.Budgerigar
}

func (s *AdminPanelTestSuite) SetupSuite() {
	s.budgerigar = stuber.NewBudgerigar()
	server, err := NewRestServer(s.T().Context(), s.budgerigar, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.server = server
}

func (s *AdminPanelTestSuite) SetupTest() {
	s.budgerigar.Clear()
}

//nolint:funlen
func (s *AdminPanelTestSuite) TestSearchStubsWithRequestInternalHeader() {
	stubData := `[{
		"service": "TestService",
		"method": "TestMethod",
		"input": {"equals": {"key": "test_value"}},
		"output": {"data": {"result": "internal_search_test"}}
	}]`

	addReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	unusedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err := json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Require().Len(unusedStubs, 1)

	searchData := `{
		"service": "TestService",
		"method": "TestMethod",
		"data": {"key": "test_value"}
	}`

	searchReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
	searchReq.Header.Set("Content-Type", "application/json")
	searchReq.Header.Set("X-Gripmock-Requestinternal", "true")

	searchW := httptest.NewRecorder()

	s.server.SearchStubs(searchW, searchReq)
	s.Equal(http.StatusOK, searchW.Code)

	var searchResult map[string]any

	err = json.Unmarshal(searchW.Body.Bytes(), &searchResult)
	s.Require().NoError(err)

	data, ok := searchResult["data"].(map[string]any)
	s.Require().True(ok, "Response should have data field")
	s.Equal("internal_search_test", data["result"])

	unusedReq2 := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/unused", nil)
	unusedW2 := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW2, unusedReq2)
	s.Require().Equal(http.StatusOK, unusedW2.Code)

	var unusedStubsAfter []*stuber.Stub

	err = json.Unmarshal(unusedW2.Body.Bytes(), &unusedStubsAfter)
	s.Require().NoError(err)
	s.Require().Len(unusedStubsAfter, 1, "Stub should still be unused after internal search")

	usedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err = json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Require().Empty(usedStubs, "No stubs should be marked as used after internal search")
}

func (s *AdminPanelTestSuite) TestSearchStubsWithoutRequestInternalHeader() {
	stubData := `[{
		"service": "TestService",
		"method": "TestMethod",
		"input": {"equals": {"key": "normal_test"}},
		"output": {"data": {"result": "normal_search_test"}}
	}]`

	addReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	searchData := `{
		"service": "TestService",
		"method": "TestMethod",
		"data": {"key": "normal_test"}
	}`

	searchReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
	searchReq.Header.Set("Content-Type", "application/json")

	searchW := httptest.NewRecorder()

	s.server.SearchStubs(searchW, searchReq)
	s.Equal(http.StatusOK, searchW.Code)

	usedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err := json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Require().Len(usedStubs, 1, "Stub should be marked as used after normal search")

	unusedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err = json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Require().Empty(unusedStubs, "No stubs should be unused after normal search")
}

//nolint:funlen
func (s *AdminPanelTestSuite) TestMultipleInternalSearches() {
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

	addReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	searches := []string{
		`{"service": "TestService", "method": "Method1", "data": {"id": 1}}`,
		`{"service": "TestService", "method": "Method2", "data": {"id": 2}}`,
		`{"service": "TestService", "method": "Method1", "data": {"id": 1}}`,
	}

	for i, searchData := range searches {
		searchReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData))
		searchReq.Header.Set("Content-Type", "application/json")
		searchReq.Header.Set("X-Gripmock-Requestinternal", "true")

		searchW := httptest.NewRecorder()

		s.server.SearchStubs(searchW, searchReq)
		s.Equal(http.StatusOK, searchW.Code, "Search %d should succeed", i+1)
	}

	unusedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err := json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Len(unusedStubs, 2, "All stubs should still be unused after multiple internal searches")

	usedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err = json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Require().Empty(usedStubs, "No stubs should be used after internal searches")
}

//nolint:funlen
func (s *AdminPanelTestSuite) TestMixedInternalAndNormalSearches() {
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

	addReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs", bytes.NewBufferString(stubData))
	addReq.Header.Set("Content-Type", "application/json")

	addW := httptest.NewRecorder()

	s.server.AddStub(addW, addReq)
	s.Require().Equal(http.StatusOK, addW.Code)

	searchData1 := `{"service": "TestService", "method": "InternalMethod", "data": {"type": "internal"}}`
	searchReq1 := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData1))
	searchReq1.Header.Set("Content-Type", "application/json")
	searchReq1.Header.Set("X-Gripmock-Requestinternal", "true")

	searchW1 := httptest.NewRecorder()

	s.server.SearchStubs(searchW1, searchReq1)
	s.Equal(http.StatusOK, searchW1.Code)

	searchData2 := `{"service": "TestService", "method": "NormalMethod", "data": {"type": "normal"}}`
	searchReq2 := httptest.NewRequestWithContext(s.T().Context(), http.MethodPost, "/api/stubs/search", bytes.NewBufferString(searchData2))
	searchReq2.Header.Set("Content-Type", "application/json")

	searchW2 := httptest.NewRecorder()

	s.server.SearchStubs(searchW2, searchReq2)
	s.Equal(http.StatusOK, searchW2.Code)

	unusedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/unused", nil)
	unusedW := httptest.NewRecorder()

	s.server.ListUnusedStubs(unusedW, unusedReq)
	s.Require().Equal(http.StatusOK, unusedW.Code)

	var unusedStubs []*stuber.Stub

	err := json.Unmarshal(unusedW.Body.Bytes(), &unusedStubs)
	s.Require().NoError(err)
	s.Len(unusedStubs, 1, "Only internal search stub should be unused")
	s.Equal("InternalMethod", unusedStubs[0].Method)

	usedReq := httptest.NewRequestWithContext(s.T().Context(), http.MethodGet, "/api/stubs/used", nil)
	usedW := httptest.NewRecorder()

	s.server.ListUsedStubs(usedW, usedReq)
	s.Require().Equal(http.StatusOK, usedW.Code)

	var usedStubs []*stuber.Stub

	err = json.Unmarshal(usedW.Body.Bytes(), &usedStubs)
	s.Require().NoError(err)
	s.Len(usedStubs, 1, "Only normal search stub should be used")
	s.Equal("NormalMethod", usedStubs[0].Method)
}

func TestAdminPanelTestSuite(t *testing.T) { //nolint:paralleltest
	suite.Run(t, new(AdminPanelTestSuite))
}
