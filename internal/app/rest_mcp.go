package app

import (
	"context"
	stderrors "errors"
	"maps"
	"net/http"
	"strings"

	"github.com/goccy/go-json"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
	"github.com/bavix/gripmock/v3/internal/infra/build"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

func (h *RestServer) MCPHandler() http.Handler {
	h.mcpHandlerOnce.Do(func() {
		h.mcpHandler = newMCPStreamableHandler(h)
	})

	return h.mcpHandler
}

const (
	debugCallDefaultLimit = 20
	debugCallHintsCap     = 4
)

func newMCPStreamableHandler(h *RestServer) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "gripmock", Version: build.Version}, nil)

	for _, tool := range mcpusecase.ListRuntimeTools() {
		name, _ := tool["name"].(string)
		description, _ := tool["description"].(string)
		inputSchema, _ := tool["inputSchema"].(map[string]any)

		if name == "" || inputSchema == nil {
			continue
		}

		server.AddTool(&mcp.Tool{
			Name:        name,
			Description: description,
			InputSchema: inputSchema,
		}, newMCPToolHandler(h, name))
	}

	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return handler
}

func newMCPToolHandler(h *RestServer, name string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: mcpInvalidArgError("arguments must be an object").Error()}
			}
		}

		args = mcpusecase.ApplySession(name, args, mcpSessionFromContext(ctx, req))

		result, err := callMCPToolDispatch(h, name, args)
		if err != nil {
			return nil, mcpJSONRPCError(name, err)
		}

		return &mcp.CallToolResult{
			Content:           []mcp.Content{&mcp.TextContent{Text: "OK"}},
			StructuredContent: result,
		}, nil
	}
}

func mcpSessionFromContext(ctx context.Context, req *mcp.CallToolRequest) string {
	if sessionID := muxmiddleware.FromContext(ctx); sessionID != "" {
		return sessionID
	}

	if req == nil || req.Extra == nil {
		return ""
	}

	return strings.TrimSpace(req.Extra.Header.Get(muxmiddleware.HeaderName))
}

func mcpJSONRPCError(toolName string, err error) error {
	data, marshalErr := json.Marshal(map[string]any{"tool": toolName})
	if marshalErr != nil {
		data = nil
	}

	if stderrors.Is(err, ErrMCPInvalidArgument) {
		return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: err.Error(), Data: data}
	}

	if stderrors.Is(err, ErrMCPToolNotFound) {
		return &jsonrpc.Error{Code: jsonrpc.CodeMethodNotFound, Message: err.Error(), Data: data}
	}

	return &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: err.Error(), Data: data}
}

func callMCPToolDispatch(h *RestServer, name string, args map[string]any) (map[string]any, error) {
	handlers := mcpToolHandlers(h)

	result, err, found := mcpusecase.DispatchTool(name, args, handlers)
	if !found {
		return nil, mcpUnknownTool(name)
	}

	return result, err
}

func mcpToolHandlers(h *RestServer) map[string]mcpusecase.ToolHandler {
	handlers := map[string]mcpusecase.ToolHandler{}

	mergeMCPToolHandlers(handlers, mcpGeneralToolHandlers(h))
	mergeMCPToolHandlers(handlers, mcpServicesToolHandlers(h))
	mergeMCPToolHandlers(handlers, mcpStubsToolHandlers(h))

	return handlers
}

func mcpGeneralToolHandlers(h *RestServer) map[string]mcpusecase.ToolHandler {
	return map[string]mcpusecase.ToolHandler{
		mcpusecase.ToolHealthLiveness:  func(toolArgs map[string]any) (map[string]any, error) { return mcpHealthLiveness(h, toolArgs) },
		mcpusecase.ToolHealthReadiness: func(toolArgs map[string]any) (map[string]any, error) { return mcpHealthReadiness(h, toolArgs) },
		mcpusecase.ToolHealthStatus:    func(toolArgs map[string]any) (map[string]any, error) { return mcpHealthStatus(h, toolArgs) },
		mcpusecase.ToolDashboard:       func(toolArgs map[string]any) (map[string]any, error) { return mcpDashboard(h, toolArgs) },
		mcpusecase.ToolOverview:        func(toolArgs map[string]any) (map[string]any, error) { return mcpDashboardOverview(h, toolArgs) },
		mcpusecase.ToolInfo:            func(toolArgs map[string]any) (map[string]any, error) { return mcpDashboardInfo(h, toolArgs) },
		mcpusecase.ToolSessionsList:    func(toolArgs map[string]any) (map[string]any, error) { return mcpSessionsList(h, toolArgs) },
		mcpusecase.ToolGripmockInfo:    func(toolArgs map[string]any) (map[string]any, error) { return mcpGripmockInfo(h, toolArgs) },
		mcpusecase.ToolReflectInfo:     func(toolArgs map[string]any) (map[string]any, error) { return mcpReflectInfo(h, toolArgs) },
		mcpusecase.ToolReflectSources:  func(toolArgs map[string]any) (map[string]any, error) { return mcpReflectSources(h, toolArgs) },
		mcpusecase.ToolDescriptorsAdd:  func(toolArgs map[string]any) (map[string]any, error) { return mcpDescriptorsAdd(h, toolArgs) },
		mcpusecase.ToolDescriptorsList: func(toolArgs map[string]any) (map[string]any, error) { return mcpDescriptorsList(h, toolArgs) },
		mcpusecase.ToolHistoryList:     func(toolArgs map[string]any) (map[string]any, error) { return mcpHistoryList(h, toolArgs) },
		mcpusecase.ToolHistoryErrors:   func(toolArgs map[string]any) (map[string]any, error) { return mcpHistoryErrors(h, toolArgs) },
		mcpusecase.ToolVerifyCalls:     func(toolArgs map[string]any) (map[string]any, error) { return mcpVerifyCalls(h, toolArgs) },
		mcpusecase.ToolDebugCall:       func(toolArgs map[string]any) (map[string]any, error) { return mcpDebugCall(h, toolArgs) },
		mcpusecase.ToolSchemaStub:      func(toolArgs map[string]any) (map[string]any, error) { return mcpSchemaStub(h, toolArgs) },
	}
}

func mcpServicesToolHandlers(h *RestServer) map[string]mcpusecase.ToolHandler {
	return map[string]mcpusecase.ToolHandler{
		mcpusecase.ToolServicesList:    func(toolArgs map[string]any) (map[string]any, error) { return mcpServicesList(h, toolArgs) },
		mcpusecase.ToolServicesGet:     func(toolArgs map[string]any) (map[string]any, error) { return mcpServicesGet(h, toolArgs) },
		mcpusecase.ToolServicesMethods: func(toolArgs map[string]any) (map[string]any, error) { return mcpServicesMethods(h, toolArgs) },
		mcpusecase.ToolServicesMethod:  func(toolArgs map[string]any) (map[string]any, error) { return mcpServicesMethod(h, toolArgs) },
		mcpusecase.ToolServicesDelete:  func(toolArgs map[string]any) (map[string]any, error) { return mcpServicesDelete(h, toolArgs) },
	}
}

func mcpStubsToolHandlers(h *RestServer) map[string]mcpusecase.ToolHandler {
	return map[string]mcpusecase.ToolHandler{
		mcpusecase.ToolStubsUpsert: func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsUpsert(h, toolArgs) },
		mcpusecase.ToolStubsList:   func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsList(h, toolArgs) },
		mcpusecase.ToolStubsGet:    func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsGet(h, toolArgs) },
		mcpusecase.ToolStubsDelete: func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsDelete(h, toolArgs) },
		mcpusecase.ToolStubsBatchDelete: func(toolArgs map[string]any) (map[string]any, error) {
			return mcpStubsBatchDelete(h, toolArgs)
		},
		mcpusecase.ToolStubsPurge:   func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsPurge(h, toolArgs) },
		mcpusecase.ToolStubsSearch:  func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsSearch(h, toolArgs) },
		mcpusecase.ToolStubsInspect: func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsInspect(h, toolArgs) },
		mcpusecase.ToolStubsUsed:    func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsUsed(h, toolArgs) },
		mcpusecase.ToolStubsUnused:  func(toolArgs map[string]any) (map[string]any, error) { return mcpStubsUnused(h, toolArgs) },
	}
}

func mergeMCPToolHandlers(dst map[string]mcpusecase.ToolHandler, src map[string]mcpusecase.ToolHandler) {
	maps.Copy(dst, src)
}
