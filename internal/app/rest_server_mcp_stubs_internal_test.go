package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func (s *RestServerTestSuite) TestMCPStubsLifecycle() {
	// Arrange
	upsert := s.mcpToolCall(s.server, 1, "stubs.upsert", map[string]any{
		"stubs": map[string]any{
			"service": "unitconverter.v1.UnitConversionService",
			"method":  "ConvertWeight",
			"input": map[string]any{
				"equals": map[string]any{"value": float64(1), "from_unit": "POUNDS", "to_unit": "KILOGRAMS"},
			},
			"output": map[string]any{"data": map[string]any{"converted_value": 0.453592}},
		},
	})
	upsertJSON := s.mcpStructuredContent(upsert)
	idsRaw, ok := upsertJSON["ids"].([]any)
	s.Require().True(ok)
	s.Require().Len(idsRaw, 1)
	id, ok := idsRaw[0].(string)
	s.Require().True(ok)

	// Act
	listed := s.mcpToolCall(s.server, 2, "stubs.list", map[string]any{"service": "unitconverter.v1.UnitConversionService"})
	listedJSON := s.mcpStructuredContent(listed)
	stubs, ok := listedJSON["stubs"].([]any)
	s.Require().True(ok)

	got := s.mcpToolCall(s.server, 3, "stubs.get", map[string]any{"id": id})
	gotJSON := s.mcpStructuredContent(got)

	deleted := s.mcpToolCall(s.server, 4, "stubs.delete", map[string]any{"id": id})
	deletedJSON := s.mcpStructuredContent(deleted)

	gotAfterDelete := s.mcpToolCall(s.server, 5, "stubs.get", map[string]any{"id": id})
	gotAfterDeleteJSON := s.mcpStructuredContent(gotAfterDelete)

	// Assert
	s.Require().Len(stubs, 1)
	s.Require().Equal(true, gotJSON["found"])
	s.Require().Equal(true, deletedJSON["deleted"])
	s.Require().Equal(false, gotAfterDeleteJSON["found"])
}

func (s *RestServerTestSuite) TestMCPStubsBatchDeleteAndPurge() {
	// Arrange
	first := s.mcpToolCall(s.server, 1, "stubs.upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "M1",
			"input":   map[string]any{"equals": map[string]any{"x": "1"}},
			"output":  map[string]any{"data": map[string]any{"ok": true}},
		},
	})
	firstJSON := s.mcpStructuredContent(first)
	firstIDs, ok := firstJSON["ids"].([]any)
	s.Require().True(ok)
	s.Require().Len(firstIDs, 1)

	second := s.mcpToolCall(s.server, 2, "stubs.upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "M2",
			"input":   map[string]any{"equals": map[string]any{"x": "2"}},
			"output":  map[string]any{"data": map[string]any{"ok": true}},
		},
	})
	secondJSON := s.mcpStructuredContent(second)
	secondIDs, ok := secondJSON["ids"].([]any)
	s.Require().True(ok)
	s.Require().Len(secondIDs, 1)

	// Act
	batch := s.mcpToolCall(s.server, 3, "stubs.batchDelete", map[string]any{
		"ids": []any{firstIDs[0], "00000000-0000-0000-0000-000000000099"},
	})
	batchJSON := s.mcpStructuredContent(batch)

	purge := s.mcpToolCall(s.server, 4, "stubs.purge", map[string]any{})
	purgeJSON := s.mcpStructuredContent(purge)

	listAfter := s.mcpToolCall(s.server, 5, "stubs.list", map[string]any{})
	listAfterJSON := s.mcpStructuredContent(listAfter)
	deletedIDs, ok := batchJSON["deletedIds"].([]any)
	s.Require().True(ok)
	notFoundIDs, ok := batchJSON["notFoundIds"].([]any)
	s.Require().True(ok)

	// Assert
	s.Require().Len(deletedIDs, 1)
	s.Require().Len(notFoundIDs, 1)
	s.Require().Equal(firstIDs[0], deletedIDs[0])
	s.Require().Equal("00000000-0000-0000-0000-000000000099", notFoundIDs[0])
	s.Require().InDelta(float64(1), purgeJSON["deletedCount"], 0)

	stubsAfter, ok := listAfterJSON["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Empty(stubsAfter)
}

func (s *RestServerTestSuite) TestMCPStubsSearch() {
	// Arrange
	s.mcpToolCall(s.server, 1, "stubs.upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "Say",
			"input":   map[string]any{"equals": map[string]any{"name": "john"}},
			"output":  map[string]any{"data": map[string]any{"message": "hello"}},
		},
	})

	// Act
	found := s.mcpToolCall(s.server, 2, "stubs.search", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "john"},
	})
	foundJSON := s.mcpStructuredContent(found)

	notFound := s.mcpToolCall(s.server, 3, "stubs.search", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "alice"},
	})
	notFoundJSON := s.mcpStructuredContent(notFound)

	// Assert
	s.Require().Equal(true, foundJSON["matched"])
	s.Require().NotEmpty(foundJSON["stubId"])
	s.Require().Equal(false, notFoundJSON["matched"])
}

func (s *RestServerTestSuite) TestMCPInfoIncludesTools() {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/mcp", nil)
	w := httptest.NewRecorder()

	// Act
	s.server.McpInfo(w, req)

	// Assert
	s.Require().Equal(http.StatusOK, w.Code)

	var body map[string]any

	err := json.Unmarshal(w.Body.Bytes(), &body)
	s.Require().NoError(err)

	tools, ok := body["tools"].([]any)
	s.Require().True(ok)
	s.Require().NotEmpty(tools)
}

func (s *RestServerTestSuite) TestMCPSchemaStub() {
	// Act
	response := s.mcpToolCall(s.server, 20, "schema.stub", map[string]any{})
	structured := s.mcpStructuredContent(response)

	// Assert
	schemaURL, ok := structured["schemaUrl"].(string)
	s.Require().True(ok)
	s.Require().Equal("https://bavix.github.io/gripmock/schema/stub.json", schemaURL)
}
