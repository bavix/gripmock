package mcp_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
)

var errDispatchBoom = errors.New("boom")

func TestDispatchTool_CallsMatchingHandler(t *testing.T) {
	t.Parallel()

	handlers := map[string]mcpusecase.ToolHandler{
		"x.tool": func(args map[string]any) (map[string]any, error) {
			return map[string]any{"echo": args["v"]}, nil
		},
	}

	result, err, found := mcpusecase.DispatchTool("x.tool", map[string]any{"v": 7}, handlers)

	require.True(t, found)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"echo": 7}, result)
}

func TestDispatchTool_ReturnsNotFoundForUnknownTool(t *testing.T) {
	t.Parallel()

	result, err, found := mcpusecase.DispatchTool("missing", nil, map[string]mcpusecase.ToolHandler{})

	require.False(t, found)
	require.NoError(t, err)
	require.Equal(t, map[string]any{}, result)
}

func TestDispatchTool_PropagatesHandlerError(t *testing.T) {
	t.Parallel()

	handlers := map[string]mcpusecase.ToolHandler{
		"x.tool": func(map[string]any) (map[string]any, error) {
			return nil, errDispatchBoom
		},
	}

	result, err, found := mcpusecase.DispatchTool("x.tool", nil, handlers)

	require.True(t, found)
	require.ErrorIs(t, err, errDispatchBoom)
	require.Nil(t, result)
}
