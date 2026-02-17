package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
)

func TestErrorResponse_UsesJSONRPCVersion(t *testing.T) {
	t.Parallel()

	id := new(rest.McpID)
	require.NoError(t, id.FromMcpID1(42))

	resp := mcpusecase.ErrorResponse(id, mcpusecase.ErrorCodeInvalidArg, "bad args", map[string]any{"tool": "x"})

	require.Equal(t, mcpusecase.JSONRPCVersion, resp.Jsonrpc)
	require.NotNil(t, resp.Error)
	require.Equal(t, mcpusecase.ErrorCodeInvalidArg, resp.Error.Code)
	require.Equal(t, "bad args", resp.Error.Message)
	require.Equal(t, map[string]any{"tool": "x"}, resp.Error.Data)
	require.Equal(t, id, resp.Id)
}

func TestParsePayloadErrorResponse_UsesParseCode(t *testing.T) {
	t.Parallel()

	resp := mcpusecase.ParsePayloadErrorResponse("invalid JSON payload")

	require.Equal(t, mcpusecase.JSONRPCVersion, resp.Jsonrpc)
	require.Nil(t, resp.Id)
	require.NotNil(t, resp.Error)
	require.Equal(t, mcpusecase.ErrorCodeParse, resp.Error.Code)
	require.Equal(t, "invalid JSON payload", resp.Error.Message)
	require.Nil(t, resp.Error.Data)
}

func TestInitializeResponse_ContainsCapabilitiesAndServerInfo(t *testing.T) {
	t.Parallel()

	id := new(rest.McpID)
	require.NoError(t, id.FromMcpID1(7))

	resp := mcpusecase.InitializeResponse(id, "v-test")

	require.Equal(t, mcpusecase.JSONRPCVersion, resp.Jsonrpc)
	require.Equal(t, id, resp.Id)
	require.Nil(t, resp.Error)

	result := resp.Result
	require.Equal(t, mcpusecase.ProtocolVersion, result["protocolVersion"])

	serverInfo, ok := result["serverInfo"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gripmock", serverInfo["name"])
	require.Equal(t, "v-test", serverInfo["version"])
}

func TestToolCallSuccessResponse_ContainsMCPContentFields(t *testing.T) {
	t.Parallel()

	id := new(rest.McpID)
	require.NoError(t, id.FromMcpID0("abc"))

	payload := map[string]any{"serviceIDs": []string{"svc.A"}}
	resp := mcpusecase.ToolCallSuccessResponse(id, payload)

	require.Equal(t, mcpusecase.JSONRPCVersion, resp.Jsonrpc)
	require.Equal(t, id, resp.Id)

	result := resp.Result
	require.Equal(t, payload, result["structuredContent"])
	require.Equal(t, false, result["isError"])

	content, ok := result["content"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, content, 1)
	require.Equal(t, "text", content[0]["type"])
	require.Equal(t, "OK", content[0]["text"])
}
