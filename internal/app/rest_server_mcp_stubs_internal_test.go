package app

func (s *RestServerTestSuite) TestMCPStubsLifecycle() {
	upsert := s.mcpToolCall(s.server, 1, "stubs_upsert", map[string]any{
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

	listed := s.mcpToolCall(s.server, 2, "stubs_list", map[string]any{"service": "unitconverter.v1.UnitConversionService"})
	listedJSON := s.mcpStructuredContent(listed)
	stubs, ok := listedJSON["stubs"].([]any)
	s.Require().True(ok)

	got := s.mcpToolCall(s.server, 3, "stubs_get", map[string]any{"id": id})
	gotJSON := s.mcpStructuredContent(got)

	deleted := s.mcpToolCall(s.server, 4, "stubs_delete", map[string]any{"id": id})
	deletedJSON := s.mcpStructuredContent(deleted)

	gotAfterDelete := s.mcpToolCall(s.server, 5, "stubs_get", map[string]any{"id": id})
	gotAfterDeleteJSON := s.mcpStructuredContent(gotAfterDelete)

	s.Require().Len(stubs, 1)
	s.Require().Equal(true, gotJSON["found"])
	s.Require().Equal(true, deletedJSON["deleted"])
	s.Require().Equal(false, gotAfterDeleteJSON["found"])
}

func (s *RestServerTestSuite) TestMCPStubsBatchDeleteAndPurge() {
	first := s.mcpToolCall(s.server, 1, "stubs_upsert", map[string]any{
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

	second := s.mcpToolCall(s.server, 2, "stubs_upsert", map[string]any{
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

	batch := s.mcpToolCall(s.server, 3, "stubs_batch_delete", map[string]any{
		"ids": []any{firstIDs[0], "00000000-0000-0000-0000-000000000099"},
	})
	batchJSON := s.mcpStructuredContent(batch)

	purge := s.mcpToolCall(s.server, 4, "stubs_purge", map[string]any{})
	purgeJSON := s.mcpStructuredContent(purge)

	listAfter := s.mcpToolCall(s.server, 5, "stubs_list", map[string]any{})
	listAfterJSON := s.mcpStructuredContent(listAfter)
	deletedIDs, ok := batchJSON["deletedIds"].([]any)
	s.Require().True(ok)
	notFoundIDs, ok := batchJSON["notFoundIds"].([]any)
	s.Require().True(ok)

	s.Require().Len(deletedIDs, 1)
	s.Require().Len(notFoundIDs, 1)
	s.Require().Equal(firstIDs[0], deletedIDs[0])
	s.Require().Equal("00000000-0000-0000-0000-000000000099", notFoundIDs[0])
	s.Require().InDelta(float64(1), purgeJSON["deletedCount"], 0)

	stubsAfter, ok := listAfterJSON["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Empty(stubsAfter)
}

func (s *RestServerTestSuite) TestMCPStubsValidate() {
	valid := s.mcpToolCall(s.server, 1, "stubs_validate", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "Say",
			"input":   map[string]any{"equals": map[string]any{"name": "john"}},
			"output":  map[string]any{"data": map[string]any{"message": "hello"}},
		},
	})
	validJSON := s.mcpStructuredContent(valid)

	invalid := s.mcpToolCall(s.server, 2, "stubs_validate", map[string]any{
		"stubs": map[string]any{
			"method": "Say",
			"output": map[string]any{"data": map[string]any{"message": "hello"}},
		},
	})

	listAfter := s.mcpToolCall(s.server, 3, "stubs_list", map[string]any{})
	listAfterJSON := s.mcpStructuredContent(listAfter)

	s.Require().Equal(true, validJSON["valid"])
	normalized, ok := validJSON["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(normalized, 1)
	first, ok := normalized[0].(map[string]any)
	s.Require().True(ok)

	_, hasID := first["id"]
	s.Require().False(hasID)

	s.Require().InDelta(float64(-32602), s.mcpErrorCode(invalid), 0)

	stubsAfter, ok := listAfterJSON["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Empty(stubsAfter)
}

func (s *RestServerTestSuite) TestMCPStubsListFilters() {
	s.mcpToolCall(s.server, 1, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service":  "alpha.Svc",
			"method":   "M",
			"priority": float64(1),
			"source":   "file",
			"input":    map[string]any{"equals": map[string]any{"x": "1"}},
			"output":   map[string]any{"data": map[string]any{"ok": true}},
		},
	})
	s.mcpToolCall(s.server, 2, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service":  "beta.Svc",
			"method":   "M",
			"priority": float64(5),
			"source":   "http",
			"input":    map[string]any{"equals": map[string]any{"x": "2"}},
			"output":   map[string]any{"data": map[string]any{"ok": true}},
		},
	})

	byQuery := s.mcpStructuredContent(s.mcpToolCall(s.server, 3, "stubs_list", map[string]any{"q": "beta"}))
	bySource := s.mcpStructuredContent(s.mcpToolCall(s.server, 4, "stubs_list", map[string]any{"source": "file"}))
	sorted := s.mcpStructuredContent(s.mcpToolCall(s.server, 5, "stubs_list", map[string]any{"sort": "service_asc"}))

	queryStubs, ok := byQuery["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(queryStubs, 1)
	s.Require().Equal("beta.Svc", mapStrField(queryStubs[0], "service"))

	sourceStubs, ok := bySource["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(sourceStubs, 1)
	s.Require().Equal("alpha.Svc", mapStrField(sourceStubs[0], "service"))

	sortedStubs, ok := sorted["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(sortedStubs, 2)
	s.Require().Equal("alpha.Svc", mapStrField(sortedStubs[0], "service"))
	s.Require().Equal("beta.Svc", mapStrField(sortedStubs[1], "service"))

	s.Require().InDelta(float64(1), byQuery["total"], 0)
	s.Require().InDelta(float64(1), bySource["total"], 0)
	s.Require().InDelta(float64(2), sorted["total"], 0)
}

func (s *RestServerTestSuite) TestMCPStubsListPagination() {
	for i, svc := range []string{"a.Svc", "b.Svc", "c.Svc"} {
		s.mcpToolCall(s.server, i+1, "stubs_upsert", map[string]any{
			"stubs": map[string]any{
				"service": svc,
				"method":  "M",
				"input":   map[string]any{"equals": map[string]any{"x": svc}},
				"output":  map[string]any{"data": map[string]any{"ok": true}},
			},
		})
	}

	page1 := s.mcpStructuredContent(s.mcpToolCall(s.server, 10, "stubs_list", map[string]any{
		"sort": "service_asc", "limit": 2,
	}))
	page2 := s.mcpStructuredContent(s.mcpToolCall(s.server, 11, "stubs_list", map[string]any{
		"sort": "service_asc", "limit": 2, "offset": 2,
	}))

	s.Require().InDelta(float64(3), page1["total"], 0)
	s.Require().InDelta(float64(3), page2["total"], 0)

	p1, ok := page1["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(p1, 2)
	s.Require().Equal("a.Svc", mapStrField(p1[0], "service"))
	s.Require().Equal("b.Svc", mapStrField(p1[1], "service"))

	p2, ok := page2["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(p2, 1)
	s.Require().Equal("c.Svc", mapStrField(p2[0], "service"))
}

func (s *RestServerTestSuite) TestMCPStubsListUsedFlag() {
	s.mcpToolCall(s.server, 1, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "Say",
			"input":   map[string]any{"equals": map[string]any{"name": "john"}},
			"output":  map[string]any{"data": map[string]any{"message": "hello"}},
		},
	})

	before := s.mcpStructuredContent(s.mcpToolCall(s.server, 2, "stubs_list", map[string]any{}))
	beforeStubs, ok := before["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(beforeStubs, 1)
	s.Require().NotEqual(true, mapField(beforeStubs[0], "used"))

	matched := s.mcpStructuredContent(s.mcpToolCall(s.server, 3, "stubs_search", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "john"},
	}))
	s.Require().Equal(true, matched["matched"])

	after := s.mcpStructuredContent(s.mcpToolCall(s.server, 4, "stubs_list", map[string]any{}))
	afterStubs, ok := after["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Len(afterStubs, 1)
	s.Require().Equal(true, mapField(afterStubs[0], "used"))

	unused := s.mcpStructuredContent(s.mcpToolCall(s.server, 5, "stubs_unused", map[string]any{}))
	unusedStubs, ok := unused["stubs"].([]any)
	s.Require().True(ok)
	s.Require().Empty(unusedStubs)
}

func (s *RestServerTestSuite) TestMCPMockCall() {
	server := s.newRestServerWithHistory()

	s.mcpToolCall(server, 1, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "Say",
			"input":   map[string]any{"equals": map[string]any{"name": "john"}},
			"output":  map[string]any{"data": map[string]any{"message": "hello {{.Request.name}}"}},
		},
	})

	called := s.mcpStructuredContent(s.mcpToolCall(server, 2, "mock_call", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "john"},
	}))

	missed := s.mcpStructuredContent(s.mcpToolCall(server, 3, "mock_call", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "alice"},
	}))

	history := s.mcpStructuredContent(s.mcpToolCall(server, 4, "history_list", map[string]any{"service": "svc"}))

	s.Require().Equal(true, called["matched"])
	s.Require().InDelta(float64(0), called["code"], 0)
	s.Require().Equal("OK", called["codeName"])
	data, ok := called["data"].(map[string]any)
	s.Require().True(ok)
	s.Require().Equal("hello john", data["message"])
	s.Require().NotEmpty(called["stubId"])

	s.Require().Equal(false, missed["matched"])

	records, ok := history["records"].([]any)
	s.Require().True(ok)
	s.Require().Len(records, 1)
	s.Require().InDelta(float64(1), history["total"], 0)
}

func (s *RestServerTestSuite) TestMCPMockCallError() {
	server := s.newRestServerWithHistory()

	s.mcpToolCall(server, 1, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "Say",
			"input":   map[string]any{"equals": map[string]any{"name": "boom"}},
			"output":  map[string]any{"error": "kaboom", "code": float64(5)},
		},
	})

	called := s.mcpStructuredContent(s.mcpToolCall(server, 2, "mock_call", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "boom"},
	}))

	s.Require().Equal(true, called["matched"])
	s.Require().InDelta(float64(5), called["code"], 0)
	s.Require().Equal("NotFound", called["codeName"])
	s.Require().Equal("kaboom", called["error"])
}

func (s *RestServerTestSuite) TestMCPMockCallTemplateError() {
	server := s.newRestServerWithHistory()

	s.mcpToolCall(server, 1, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "Say",
			"input":   map[string]any{"equals": map[string]any{"name": "john"}},
			"output":  map[string]any{"data": map[string]any{"message": "hi {{nope}}"}},
		},
	})

	called := s.mcpStructuredContent(s.mcpToolCall(server, 2, "mock_call", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "john"},
	}))

	s.Require().Equal(true, called["matched"])
	s.Require().InDelta(float64(13), called["code"], 0)
	s.Require().Equal("Internal", called["codeName"])
	s.Require().Contains(called["error"], "failed to process templates")
	_, hasData := called["data"]
	s.Require().False(hasData)
}

func (s *RestServerTestSuite) TestMCPStubsSearch() {
	s.mcpToolCall(s.server, 1, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service": "svc",
			"method":  "Say",
			"input":   map[string]any{"equals": map[string]any{"name": "john"}},
			"output":  map[string]any{"data": map[string]any{"message": "hello"}},
		},
	})

	found := s.mcpToolCall(s.server, 2, "stubs_search", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "john"},
	})
	foundJSON := s.mcpStructuredContent(found)

	notFound := s.mcpToolCall(s.server, 3, "stubs_search", map[string]any{
		"service": "svc",
		"method":  "Say",
		"payload": map[string]any{"name": "alice"},
	})
	notFoundJSON := s.mcpStructuredContent(notFound)

	s.Require().Equal(true, foundJSON["matched"])
	s.Require().NotEmpty(foundJSON["stubId"])
	s.Require().Equal(false, notFoundJSON["matched"])
}

func (s *RestServerTestSuite) TestMCPInfoIncludesTools() {
	response := s.mcpCallOK(s.server, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	result, ok := response["result"].(map[string]any)
	s.Require().True(ok)

	tools, ok := result["tools"].([]any)
	s.Require().True(ok)
	s.Require().NotEmpty(tools)
}

func (s *RestServerTestSuite) TestMCPSchemaStub() {
	response := s.mcpToolCall(s.server, 20, "schema_stub", map[string]any{})
	structured := s.mcpStructuredContent(response)

	schemaURL, ok := structured["schemaUrl"].(string)
	s.Require().True(ok)
	s.Require().Equal("https://bavix.github.io/gripmock/schema/stub.json", schemaURL)
}
