package app

import (
	"context"
	stderrors "errors"
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

// mcpToolFunc is the shared shape of every MCP tool implementation.
type mcpToolFunc func(*RestServer, map[string]any) (map[string]any, error)

// bindTool adapts an (h, args) tool function to the arg-only ToolHandler the
// dispatcher expects, capturing the server.
func bindTool(h *RestServer, fn mcpToolFunc) mcpusecase.ToolHandler {
	return func(args map[string]any) (map[string]any, error) {
		return fn(h, args)
	}
}

func mcpToolHandlers(h *RestServer) map[string]mcpusecase.ToolHandler {
	funcs := map[string]mcpToolFunc{
		// general
		mcpusecase.ToolHealthLiveness:  mcpHealthLiveness,
		mcpusecase.ToolHealthReadiness: mcpHealthReadiness,
		mcpusecase.ToolHealthStatus:    mcpHealthStatus,
		mcpusecase.ToolDashboard:       mcpDashboard,
		mcpusecase.ToolOverview:        mcpDashboardOverview,
		mcpusecase.ToolInfo:            mcpDashboardInfo,
		mcpusecase.ToolSessionsList:    mcpSessionsList,
		mcpusecase.ToolGripmockInfo:    mcpGripmockInfo,
		mcpusecase.ToolReflectInfo:     mcpReflectInfo,
		mcpusecase.ToolReflectSources:  mcpReflectSources,
		mcpusecase.ToolDescriptorsAdd:  mcpDescriptorsAdd,
		mcpusecase.ToolDescriptorsList: mcpDescriptorsList,
		mcpusecase.ToolHistoryList:     mcpHistoryList,
		mcpusecase.ToolHistoryErrors:   mcpHistoryErrors,
		mcpusecase.ToolVerifyCalls:     mcpVerifyCalls,
		mcpusecase.ToolDebugCall:       mcpDebugCall,
		mcpusecase.ToolSchemaStub:      mcpSchemaStub,
		// services
		mcpusecase.ToolServicesList:    mcpServicesList,
		mcpusecase.ToolServicesGet:     mcpServicesGet,
		mcpusecase.ToolServicesMethods: mcpServicesMethods,
		mcpusecase.ToolServicesMethod:  mcpServicesMethod,
		mcpusecase.ToolServicesDelete:  mcpServicesDelete,
		// stubs
		mcpusecase.ToolStubsUpsert:      mcpStubsUpsert,
		mcpusecase.ToolStubsValidate:    mcpStubsValidate,
		mcpusecase.ToolStubsList:        mcpStubsList,
		mcpusecase.ToolStubsGet:         mcpStubsGet,
		mcpusecase.ToolStubsDelete:      mcpStubsDelete,
		mcpusecase.ToolStubsBatchDelete: mcpStubsBatchDelete,
		mcpusecase.ToolStubsPurge:       mcpStubsPurge,
		mcpusecase.ToolStubsSearch:      mcpStubsSearch,
		mcpusecase.ToolStubsInspect:     mcpStubsInspect,
		mcpusecase.ToolStubsUsed:        mcpStubsUsed,
		mcpusecase.ToolStubsUnused:      mcpStubsUnused,
		mcpusecase.ToolMockCall:         mcpMockCall,
	}

	handlers := make(map[string]mcpusecase.ToolHandler, len(funcs))
	for name, fn := range funcs {
		handlers[name] = bindTool(h, fn)
	}

	return handlers
}
