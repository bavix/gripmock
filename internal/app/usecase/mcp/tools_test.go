package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
)

func TestListTools_ContainsAllExpectedTools(t *testing.T) {
	t.Parallel()

	// Arrange
	expected := map[string]struct{}{
		mcpusecase.ToolDescriptorsAdd:  {},
		mcpusecase.ToolDescriptorsList: {},
		mcpusecase.ToolServicesList:    {},
		mcpusecase.ToolServicesDelete:  {},
		mcpusecase.ToolHistoryList:     {},
		mcpusecase.ToolHistoryErrors:   {},
		mcpusecase.ToolDebugCall:       {},
		mcpusecase.ToolSchemaStub:      {},
	}

	// Act
	tools := mcpusecase.ListTools()

	// Assert
	require.Len(t, tools, len(expected))

	for _, tool := range tools {
		name, ok := tool["name"].(string)
		require.True(t, ok)

		_, found := expected[name]
		require.True(t, found)

		schema, ok := tool["inputSchema"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "object", schema["type"])
	}
}

func TestListTools_DebugCallRequiresService(t *testing.T) {
	t.Parallel()

	// Act
	tools := mcpusecase.ListTools()

	// Assert
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name != mcpusecase.ToolDebugCall {
			continue
		}

		schema, ok := tool["inputSchema"].(map[string]any)
		require.True(t, ok)

		required, ok := schema["required"].([]string)
		require.True(t, ok)
		require.Equal(t, []string{"service"}, required)

		return
	}

	t.Fatal("debug.call tool not found")
}
