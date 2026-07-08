package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
)

func TestToolUsesSession(t *testing.T) {
	t.Parallel()

	tools := []string{"history_list", "history_errors", "debug_call"}

	for _, tool := range tools {
		usesSession := mcpusecase.ToolUsesSession(tool)
		require.True(t, usesSession)
	}
}

func TestToolUsesSessionFalseForOtherTools(t *testing.T) {
	t.Parallel()

	usesSession := mcpusecase.ToolUsesSession("services_list")

	require.False(t, usesSession)
}
