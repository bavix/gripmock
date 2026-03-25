package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
)

func TestListToolsContainsAllExpectedTools(t *testing.T) {
	t.Parallel()

	// Arrange
	expected := map[string]struct{}{
		mcpusecase.ToolHealthLiveness:  {},
		mcpusecase.ToolHealthReadiness: {},
		mcpusecase.ToolHealthStatus:    {},
		mcpusecase.ToolDashboard:       {},
		mcpusecase.ToolOverview:        {},
		mcpusecase.ToolInfo:            {},
		mcpusecase.ToolSessionsList:    {},
		mcpusecase.ToolGripmockInfo:    {},
		mcpusecase.ToolReflectInfo:     {},
		mcpusecase.ToolReflectSources:  {},
		mcpusecase.ToolDescriptorsAdd:  {},
		mcpusecase.ToolDescriptorsList: {},
		mcpusecase.ToolServicesList:    {},
		mcpusecase.ToolServicesGet:     {},
		mcpusecase.ToolServicesMethods: {},
		mcpusecase.ToolServicesMethod:  {},
		mcpusecase.ToolServicesDelete:  {},
		mcpusecase.ToolHistoryList:     {},
		mcpusecase.ToolHistoryErrors:   {},
		mcpusecase.ToolVerifyCalls:     {},
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

func TestListToolsDebugCallRequiresService(t *testing.T) {
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

func TestListRuntimeToolsContainsStubAndInspectTools(t *testing.T) {
	t.Parallel()

	tools := mcpusecase.ListRuntimeTools()

	expected := map[string]struct{}{
		mcpusecase.ToolStubsUpsert:      {},
		mcpusecase.ToolStubsList:        {},
		mcpusecase.ToolStubsGet:         {},
		mcpusecase.ToolStubsDelete:      {},
		mcpusecase.ToolStubsBatchDelete: {},
		mcpusecase.ToolStubsPurge:       {},
		mcpusecase.ToolStubsSearch:      {},
		mcpusecase.ToolStubsInspect:     {},
		mcpusecase.ToolStubsUsed:        {},
		mcpusecase.ToolStubsUnused:      {},
	}

	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		seen[name] = struct{}{}
	}

	for tool := range expected {
		_, ok := seen[tool]
		require.True(t, ok, "runtime tools should contain %s", tool)
	}
}

func TestListRuntimeToolsHasUniqueNames(t *testing.T) {
	t.Parallel()

	tools := mcpusecase.ListRuntimeTools()
	seen := make(map[string]struct{}, len(tools))

	for _, tool := range tools {
		name, ok := tool["name"].(string)
		require.True(t, ok)
		require.NotEmpty(t, name)

		_, duplicate := seen[name]
		require.False(t, duplicate, "duplicate runtime tool: %s", name)
		seen[name] = struct{}{}
	}
}
