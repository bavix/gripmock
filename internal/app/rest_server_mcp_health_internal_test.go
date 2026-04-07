package app

func (s *RestServerTestSuite) TestMCPStubsUpsertAllowsProtectedHealthServiceDefinition() {
	// Arrange
	payload := map[string]any{
		"stubs": map[string]any{
			"service": "grpc.health.v1.Health",
			"method":  "Check",
			"input": map[string]any{
				"equals": map[string]any{"service": "gripmock"},
			},
			"output": map[string]any{
				"data": map[string]any{"status": "NOT_SERVING"},
			},
		},
	}

	// Act
	response := s.mcpToolCall(s.server, 1001, "stubs.upsert", payload)

	// Assert
	structured := s.mcpStructuredContent(response)
	ids, ok := structured["ids"].([]any)
	s.Require().True(ok)
	s.Require().Len(ids, 1)
}
