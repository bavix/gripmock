package mcp

import "github.com/bavix/gripmock/v3/internal/domain/rest"

const (
	ProtocolVersion = "2024-11-05"
	JSONRPCVersion  = "2.0"

	ErrorCodeParse      = -32700
	ErrorCodeInvalidReq = -32600
	ErrorCodeNotFound   = -32601
	ErrorCodeInvalidArg = -32602
	ErrorCodeInternal   = -32603
)

func ErrorResponse(id *rest.McpID, code int, message string, data map[string]any) rest.McpResponse {
	return rest.McpResponse{
		Jsonrpc: JSONRPCVersion,
		Id:      id,
		Error: &rest.McpError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func ParsePayloadErrorResponse(message string) rest.McpResponse {
	return ErrorResponse(nil, ErrorCodeParse, message, nil)
}

func InitializeResponse(id *rest.McpID, serverVersion string) rest.McpResponse {
	return rest.McpResponse{
		Jsonrpc: JSONRPCVersion,
		Id:      id,
		Result: map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]any{
				"name":    "gripmock",
				"version": serverVersion,
			},
		},
	}
}

func PingResponse(id *rest.McpID) rest.McpResponse {
	return rest.McpResponse{Jsonrpc: JSONRPCVersion, Id: id, Result: map[string]any{}}
}

func ToolsListResponse(id *rest.McpID, tools []map[string]any) rest.McpResponse {
	return rest.McpResponse{Jsonrpc: JSONRPCVersion, Id: id, Result: map[string]any{"tools": tools}}
}

func ToolCallSuccessResponse(id *rest.McpID, result map[string]any) rest.McpResponse {
	return rest.McpResponse{
		Jsonrpc: JSONRPCVersion,
		Id:      id,
		Result: map[string]any{
			"content": []map[string]any{{
				"type": "text",
				"text": "OK",
			}},
			"structuredContent": result,
			"isError":           false,
		},
	}
}
