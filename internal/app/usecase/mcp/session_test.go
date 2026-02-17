package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
)

func TestToolUsesSession(t *testing.T) {
	t.Parallel()

	// Arrange
	tools := []string{"history.list", "history.errors", "debug.call"}

	for _, tool := range tools {
		// Act
		usesSession := mcpusecase.ToolUsesSession(tool)

		// Assert
		require.True(t, usesSession)
	}
}

func TestToolUsesSession_FalseForOtherTools(t *testing.T) {
	t.Parallel()

	// Act
	usesSession := mcpusecase.ToolUsesSession("services.list")

	// Assert
	require.False(t, usesSession)
}

func TestApplyTransportSession_InjectsHeaderSession(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodPost, "/api/mcp", nil)
	req.Header.Set("X-Gripmock-Session", "A")

	// Act
	args := mcpusecase.ApplyTransportSession(req, "history.list", map[string]any{"service": "svc"})

	// Assert
	require.Equal(t, "A", args["session"])
}

func TestApplyTransportSession_DoesNotOverrideExplicitSession(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodPost, "/api/mcp", nil)
	req.Header.Set("X-Gripmock-Session", "A")

	// Act
	args := mcpusecase.ApplyTransportSession(req, "history.list", map[string]any{"session": "B"})

	// Assert
	require.Equal(t, "B", args["session"])
}

func TestApplyTransportSession_SkipsUnsupportedTool(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodPost, "/api/mcp", nil)
	req.Header.Set("X-Gripmock-Session", "A")

	// Act
	args := mcpusecase.ApplyTransportSession(req, "services.list", map[string]any{"x": 1})

	// Assert
	_, hasSession := args["session"]
	require.False(t, hasSession)
}
