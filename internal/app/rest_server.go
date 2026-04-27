package app

import (
	"context"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"maps"
	"net/http"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/build"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Extender defines the interface for extending stub functionality.
type Extender interface {
	Wait(ctx context.Context)
}

// RestServer handles HTTP REST API requests for stub management.
type RestServer struct {
	ok              atomic.Bool
	startedAt       time.Time
	descriptorOpsMu sync.Mutex
	mcpHandlerOnce  sync.Once
	budgerigar      *stuber.Budgerigar
	history         history.Reader
	validator       *validator.Validate
	restDescriptors *descriptors.Registry
	mcpHandler      http.Handler
}

var _ rest.ServerInterface = &RestServer{}

// NewRestServer creates a new REST server instance with the specified dependencies.
// If historyReader is nil, /api/history and /api/verify return empty/error.
// If stubValidator is nil, a new default validator is created automatically.
func NewRestServer(
	ctx context.Context,
	budgerigar *stuber.Budgerigar,
	extender Extender,
	historyReader history.Reader,
	stubValidator *validator.Validate,
	registry *descriptors.Registry,
) (*RestServer, error) {
	v := stubValidator
	if v == nil {
		var err error

		v, err = NewStubValidator()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create stub validator")
		}
	}

	r := registry
	if r == nil {
		r = descriptors.NewRegistry()
	}

	server := &RestServer{
		startedAt:       time.Now(),
		budgerigar:      budgerigar,
		history:         historyReader,
		validator:       v,
		restDescriptors: r,
	}

	go func() {
		if extender != nil {
			extender.Wait(ctx)
		}

		server.ok.Store(true)
	}()

	return server, nil
}

const (
	servicesListCap   = 16
	serviceMethodsCap = 32
	stubSchemaURL     = "https://bavix.github.io/gripmock/schema/stub.json"
)

var (
	errServiceNotFound = stderrors.New("service not found")
	errMethodNotFound  = stderrors.New("method not found in service")
)

// ServicesList returns a list of all available gRPC services (startup + REST-added).
func (h *RestServer) ServicesList(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.collectAllServices())
}

func splitLast(s string, sep string) []string {
	lastDot := strings.LastIndex(s, sep)
	if lastDot == -1 {
		return []string{s, ""}
	}

	return []string{s[:lastDot], s[lastDot+1:]}
}

// ServiceMethodsList returns a list of methods for the specified service.
func (h *RestServer) ServiceMethodsList(w http.ResponseWriter, r *http.Request, serviceID string) {
	serviceDescriptor, ok := h.findServiceDescriptor(serviceID)
	if !ok {
		h.writeResponse(r.Context(), w, []rest.Method{})

		return
	}

	h.writeResponse(r.Context(), w, h.serviceFromDescriptor(serviceDescriptor, false).Methods)
}

// ServiceGet returns exact service metadata by id.
func (h *RestServer) ServiceGet(w http.ResponseWriter, r *http.Request, serviceID string) {
	service, ok := h.findServiceDetailed(serviceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, fmt.Errorf("%w: %s", errServiceNotFound, serviceID))

		return
	}

	h.writeResponse(r.Context(), w, service)
}

// ServiceMethodGet returns exact method metadata by service and method id.
func (h *RestServer) ServiceMethodGet(w http.ResponseWriter, r *http.Request, serviceID string, methodID string) {
	service, ok := h.findServiceDetailed(serviceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, fmt.Errorf("%w: %s", errServiceNotFound, serviceID))

		return
	}

	for _, method := range service.Methods {
		if method.Id == methodID || method.Name == methodID {
			h.writeResponse(r.Context(), w, method)

			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	h.writeResponseError(
		r.Context(),
		w,
		fmt.Errorf("%w %s in service %s", errMethodNotFound, methodID, serviceID),
	)
}

// FindByID returns a stub by ID.
func (h *RestServer) FindByID(w http.ResponseWriter, r *http.Request, uuid rest.ID) {
	stub := h.budgerigar.FindByID(uuid)
	if stub == nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponse(r.Context(), w, map[string]string{
			"error": fmt.Sprintf("Stub with ID '%s' not found", uuid),
		})

		return
	}

	h.writeResponse(r.Context(), w, stub)
}

// Readiness handles the readiness probe endpoint.
func (h *RestServer) Readiness(w http.ResponseWriter, r *http.Request) {
	if !h.ok.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		h.writeResponse(r.Context(), w, rest.MessageOK{Message: "not ready", Time: time.Now()})

		return
	}

	h.liveness(r.Context(), w)
}

// Liveness handles the liveness probe endpoint.
func (h *RestServer) Liveness(w http.ResponseWriter, r *http.Request) {
	h.liveness(r.Context(), w)
}

// DashboardOverview returns aggregated lightweight metrics for admin dashboard.
func (h *RestServer) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	payload := h.dashboardPayload(r)

	response := rest.DashboardOverview{
		TotalServices:      payload.TotalServices,
		TotalStubs:         payload.TotalStubs,
		UsedStubs:          payload.UsedStubs,
		UnusedStubs:        payload.UnusedStubs,
		TotalSessions:      payload.TotalSessions,
		RuntimeDescriptors: payload.RuntimeDescriptors,
		TotalHistory:       payload.TotalHistory,
		HistoryErrors:      payload.HistoryErrors,
	}

	h.writeResponse(r.Context(), w, response)
}

// Dashboard returns combined counters and runtime metadata for dashboard page.
func (h *RestServer) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.dashboardPayload(r))
}

// SessionsList returns distinct non-empty session IDs for UI selectors.
func (h *RestServer) SessionsList(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, rest.Sessions{Sessions: h.budgerigar.Sessions()})
}

// DashboardInfo returns build metadata and runtime process information.
func (h *RestServer) DashboardInfo(w http.ResponseWriter, r *http.Request) {
	payload := h.dashboardPayload(r)

	h.writeResponse(r.Context(), w, rest.DashboardInfo{
		AppName:            payload.AppName,
		Version:            payload.Version,
		GoVersion:          payload.GoVersion,
		Compiler:           payload.Compiler,
		Goos:               payload.Goos,
		Goarch:             payload.Goarch,
		NumCPU:             payload.NumCPU,
		StartedAt:          payload.StartedAt,
		UptimeSeconds:      payload.UptimeSeconds,
		Ready:              payload.Ready,
		HistoryEnabled:     payload.HistoryEnabled,
		TotalServices:      payload.TotalServices,
		TotalStubs:         payload.TotalStubs,
		TotalSessions:      payload.TotalSessions,
		RuntimeDescriptors: payload.RuntimeDescriptors,
	})
}

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

func mcpSchemaStub(_ *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"schemaUrl": stubSchemaURL}, nil
}

func mcpHealthLiveness(_ *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"message": "ok", "time": time.Now()}, nil
}

func mcpHealthReadiness(h *RestServer, _ map[string]any) (map[string]any, error) {
	ready := h.ok.Load()
	if !ready {
		return map[string]any{"ready": false, "message": "not ready", "time": time.Now()}, nil
	}

	return map[string]any{"ready": true, "message": "ok", "time": time.Now()}, nil
}

func mcpHealthStatus(h *RestServer, _ map[string]any) (map[string]any, error) {
	ready := h.ok.Load()

	readiness := "ok"
	if !ready {
		readiness = "not ready"
	}

	return map[string]any{
		"liveness":  "ok",
		"readiness": readiness,
		"ready":     ready,
		"time":      time.Now(),
	}, nil
}

func mcpDashboard(h *RestServer, args map[string]any) (map[string]any, error) {
	return map[string]any{"dashboard": h.dashboardPayload(mcpSessionRequest(args))}, nil
}

func mcpDashboardOverview(h *RestServer, args map[string]any) (map[string]any, error) {
	payload := h.dashboardPayload(mcpSessionRequest(args))

	return map[string]any{"overview": rest.DashboardOverview{
		TotalServices:      payload.TotalServices,
		TotalStubs:         payload.TotalStubs,
		UsedStubs:          payload.UsedStubs,
		UnusedStubs:        payload.UnusedStubs,
		TotalSessions:      payload.TotalSessions,
		RuntimeDescriptors: payload.RuntimeDescriptors,
		TotalHistory:       payload.TotalHistory,
		HistoryErrors:      payload.HistoryErrors,
	}}, nil
}

func mcpDashboardInfo(h *RestServer, args map[string]any) (map[string]any, error) {
	payload := h.dashboardPayload(mcpSessionRequest(args))

	return map[string]any{"info": rest.DashboardInfo{
		AppName:            payload.AppName,
		Version:            payload.Version,
		GoVersion:          payload.GoVersion,
		Compiler:           payload.Compiler,
		Goos:               payload.Goos,
		Goarch:             payload.Goarch,
		NumCPU:             payload.NumCPU,
		StartedAt:          payload.StartedAt,
		UptimeSeconds:      payload.UptimeSeconds,
		Ready:              payload.Ready,
		HistoryEnabled:     payload.HistoryEnabled,
		TotalServices:      payload.TotalServices,
		TotalStubs:         payload.TotalStubs,
		TotalSessions:      payload.TotalSessions,
		RuntimeDescriptors: payload.RuntimeDescriptors,
	}}, nil
}

func mcpSessionsList(h *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"sessions": h.budgerigar.Sessions()}, nil
}

func mcpGripmockInfo(h *RestServer, _ map[string]any) (map[string]any, error) {
	overview := h.dashboardPayload(nil)

	return map[string]any{
		"appName":            overview.AppName,
		"version":            overview.Version,
		"protocolVersion":    mcpusecase.ProtocolVersion,
		"historyEnabled":     overview.HistoryEnabled,
		"ready":              overview.Ready,
		"totalServices":      overview.TotalServices,
		"totalStubs":         overview.TotalStubs,
		"totalSessions":      overview.TotalSessions,
		"runtimeDescriptors": overview.RuntimeDescriptors,
		"tools":              mcpusecase.ListRuntimeTools(),
	}, nil
}

func mcpReflectInfo(h *RestServer, _ map[string]any) (map[string]any, error) {
	runtimePaths, reflectionPrefixes, reflectionFiles := runtimeDescriptorStats(h)

	globalCount := 0

	protoregistry.GlobalFiles.RangeFiles(func(_ protoreflect.FileDescriptor) bool {
		globalCount++

		return true
	})

	return map[string]any{
		"runtimeDescriptorFiles":    len(runtimePaths),
		"reflectionDescriptorFiles": reflectionFiles,
		"dynamicDescriptorFiles":    len(runtimePaths) - reflectionFiles,
		"reflectionDetected":        reflectionFiles > 0,
		"reflectionSources":         reflectionPrefixes,
		"globalDescriptorFiles":     globalCount,
	}, nil
}

func mcpReflectSources(h *RestServer, args map[string]any) (map[string]any, error) {
	runtimePaths, reflectionPrefixes, _ := runtimeDescriptorStats(h)
	reflectionPaths, dynamicPaths, _ := splitRuntimeDescriptorPaths(runtimePaths)

	kind, _ := args["kind"].(string)
	if kind == "" {
		kind = "all"
	}

	if kind != "all" && kind != "reflection" && kind != "dynamic" {
		return nil, mcpInvalidArgError("kind must be one of: all, reflection, dynamic")
	}

	offset, err := mcpIntArg(args, "offset", 0)
	if err != nil {
		return nil, err
	}

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	filtered := runtimePaths

	switch kind {
	case "reflection":
		filtered = reflectionPaths
	case "dynamic":
		filtered = dynamicPaths
	}

	total := len(filtered)
	filtered = paginateStringSlice(filtered, offset, limit)

	return map[string]any{
		"kind":   kind,
		"paths":  filtered,
		"total":  total,
		"offset": offset,
		"limit":  limit,
		"groups": map[string]any{
			"reflection": map[string]any{"count": len(reflectionPaths)},
			"dynamic":    map[string]any{"count": len(dynamicPaths)},
		},
		"reflectionSources": reflectionPrefixes,
	}, nil
}

func runtimeDescriptorStats(h *RestServer) ([]string, []string, int) {
	runtimePaths := h.restDescriptors.Paths()
	reflectionPaths, _, prefixes := splitRuntimeDescriptorPaths(runtimePaths)

	return runtimePaths, prefixes, len(reflectionPaths)
}

func splitRuntimeDescriptorPaths(runtimePaths []string) ([]string, []string, []string) {
	reflectionPrefixes := make(map[string]struct{})
	reflectionPaths := make([]string, 0, len(runtimePaths))
	dynamicPaths := make([]string, 0, len(runtimePaths))

	for _, path := range runtimePaths {
		if !strings.HasPrefix(path, "grpc_reflect_") {
			dynamicPaths = append(dynamicPaths, path)

			continue
		}

		reflectionPaths = append(reflectionPaths, path)

		prefix := path
		if idx := strings.Index(prefix, "/"); idx > 0 {
			prefix = prefix[:idx]
		}

		reflectionPrefixes[prefix] = struct{}{}
	}

	prefixes := make([]string, 0, len(reflectionPrefixes))
	for prefix := range reflectionPrefixes {
		prefixes = append(prefixes, prefix)
	}

	sort.Strings(prefixes)

	return reflectionPaths, dynamicPaths, prefixes
}

func paginateStringSlice(items []string, offset int, limit int) []string {
	if offset >= len(items) {
		return []string{}
	}

	items = items[offset:]

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items
}

func mcpSessionRequest(args map[string]any) *http.Request {
	req := &http.Request{Header: make(http.Header)}
	if sessionID, _ := args["session"].(string); sessionID != "" {
		req.Header.Set(muxmiddleware.HeaderName, sessionID)
	}

	return req
}

func mcpDescriptorsAdd(h *RestServer, args map[string]any) (map[string]any, error) {
	descriptorSetBase64, _ := args["descriptorSetBase64"].(string)
	if descriptorSetBase64 == "" {
		return nil, mcpRequiredArgError("descriptorSetBase64")
	}

	payload, err := base64.StdEncoding.DecodeString(descriptorSetBase64)
	if err != nil {
		return nil, mcpDescriptorSetBase64ArgError(err)
	}

	serviceIDs, err := registerDescriptorBytes(h, payload)
	if err != nil {
		return nil, mcpDescriptorRegistrationArgError(err)
	}

	return map[string]any{"serviceIDs": serviceIDs}, nil
}

func mcpDescriptorsList(h *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"serviceIDs": h.restDescriptors.ServiceIDs()}, nil
}

func mcpServicesList(h *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"services": h.collectAllServices()}, nil
}

func mcpServicesDelete(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	removed := unregisterService(h, serviceID)

	return map[string]any{"removed": removed > 0, "serviceID": serviceID}, nil
}

func mcpServicesGet(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	service, ok := h.findServiceDetailed(serviceID)
	if !ok {
		return nil, mcpInvalidArgError(errServiceNotFound.Error() + ": " + serviceID)
	}

	return map[string]any{"service": service}, nil
}

func mcpServicesMethods(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	serviceDescriptor, ok := h.findServiceDescriptor(serviceID)
	if !ok {
		return nil, mcpInvalidArgError(errServiceNotFound.Error() + ": " + serviceID)
	}

	return map[string]any{"methods": h.serviceFromDescriptor(serviceDescriptor, false).Methods}, nil
}

func mcpServicesMethod(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	methodID, _ := args["methodID"].(string)
	if methodID == "" {
		return nil, mcpRequiredArgError("methodID")
	}

	service, ok := h.findServiceDetailed(serviceID)
	if !ok {
		return nil, mcpInvalidArgError(errServiceNotFound.Error() + ": " + serviceID)
	}

	for _, method := range service.Methods {
		if method.Id == methodID || method.Name == methodID {
			return map[string]any{"method": method}, nil
		}
	}

	return nil, mcpInvalidArgError(errMethodNotFound.Error() + " " + methodID + " in service " + serviceID)
}

func mcpHistoryList(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	method, _ := args["method"].(string)
	session, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	records := filterHistory(h, history.FilterOpts{
		Service: service,
		Method:  method,
		Session: session,
	}, limit)

	return map[string]any{"records": records}, nil
}

func mcpHistoryErrors(h *RestServer, args map[string]any) (map[string]any, error) {
	session, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	errorsOnly := extractErrorRecords(filterHistory(h, history.FilterOpts{Session: session}, 0))
	if limit > 0 && len(errorsOnly) > limit {
		errorsOnly = errorsOnly[len(errorsOnly)-limit:]
	}

	return map[string]any{"records": errorsOnly}, nil
}

func mcpVerifyCalls(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return nil, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	if method == "" {
		return nil, mcpRequiredArgError("method")
	}

	expectedCount, err := mcpIntArg(args, "expectedCount", -1)
	if err != nil {
		return nil, err
	}

	if expectedCount < 0 {
		return nil, mcpRequiredArgError("expectedCount")
	}

	if h.history == nil {
		return map[string]any{"verified": false, "message": "history is disabled", "expected": expectedCount, "actual": 0}, nil
	}

	session, _ := args["session"].(string)
	calls := h.history.Filter(history.FilterOpts{Service: service, Method: method, Session: session})
	actual := len(calls)

	if actual != expectedCount {
		return map[string]any{
			"verified": false,
			"message":  fmt.Sprintf("expected %s/%s to be called %d times, got %d", service, method, expectedCount, actual),
			"expected": expectedCount,
			"actual":   actual,
		}, nil
	}

	return map[string]any{"verified": true, "message": "ok", "expected": expectedCount, "actual": actual}, nil
}

func mcpDebugCall(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return nil, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	session, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", debugCallDefaultLimit)
	if err != nil {
		return nil, err
	}

	stubsLimit, err := mcpIntArg(args, "stubsLimit", debugCallDefaultLimit)
	if err != nil {
		return nil, err
	}

	return debugCall(h, service, method, session, limit, stubsLimit), nil
}

func mcpStubsUpsert(h *RestServer, args map[string]any) (map[string]any, error) {
	rawStubs, ok := args["stubs"]
	if !ok || rawStubs == nil {
		return nil, mcpRequiredArgError("stubs")
	}

	stubs, err := decodeMCPStubsArg(rawStubs)
	if err != nil {
		return nil, err
	}

	sessionID, _ := args["session"].(string)

	for _, stub := range stubs {
		stub.Session = sessionID

		if err = h.validateStub(stub); err != nil {
			return nil, mcpInvalidArgErrorWithCause(err.Error(), err)
		}
	}

	ids := h.budgerigar.PutMany(stubs...)

	return map[string]any{"ids": uuidListToStringSlice(ids)}, nil
}

func mcpStubsList(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := listMCPStubs(h.budgerigar.All(), args)
	if err != nil {
		return nil, err
	}

	return map[string]any{"stubs": stubs}, nil
}

func mcpStubsUsed(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := listMCPStubs(h.budgerigar.Used(), args)
	if err != nil {
		return nil, err
	}

	return map[string]any{"stubs": stubs}, nil
}

func mcpStubsUnused(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := listMCPStubs(h.budgerigar.Unused(), args)
	if err != nil {
		return nil, err
	}

	return map[string]any{"stubs": stubs}, nil
}

func mcpStubsGet(h *RestServer, args map[string]any) (map[string]any, error) {
	id, err := mcpUUIDArg(args, "id")
	if err != nil {
		return nil, err
	}

	found := h.budgerigar.FindByID(id)

	if found == nil {
		return map[string]any{"found": false, "id": id.String()}, nil
	}

	return map[string]any{"found": true, "stub": found}, nil
}

func mcpStubsDelete(h *RestServer, args map[string]any) (map[string]any, error) {
	id, err := mcpUUIDArg(args, "id")
	if err != nil {
		return nil, err
	}

	deleted := h.budgerigar.DeleteByID(id) > 0

	return map[string]any{"deleted": deleted, "id": id.String()}, nil
}

func mcpStubsBatchDelete(h *RestServer, args map[string]any) (map[string]any, error) {
	idStrings, err := mcpStringSliceArg(args, "ids")
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(idStrings))
	deletedIDs := make([]string, 0, len(idStrings))
	notFoundIDs := make([]string, 0)

	for _, idString := range idStrings {
		id, parseErr := uuid.Parse(idString)
		if parseErr != nil {
			return nil, mcpUUIDArgError("ids", idString, parseErr)
		}

		ids = append(ids, id)

		if h.budgerigar.FindByID(id) == nil {
			notFoundIDs = append(notFoundIDs, idString)
		} else {
			deletedIDs = append(deletedIDs, idString)
		}
	}

	if len(ids) > 0 {
		h.budgerigar.DeleteByID(ids...)
	}

	return map[string]any{
		"deletedIds":  deletedIDs,
		"notFoundIds": notFoundIDs,
	}, nil
}

func mcpStubsPurge(h *RestServer, args map[string]any) (map[string]any, error) {
	sessionID, _ := args["session"].(string)
	if sessionID != "" {
		deletedCount := h.budgerigar.DeleteSession(sessionID)

		return map[string]any{"deletedCount": deletedCount, "session": sessionID}, nil
	}

	deletedCount := len(h.budgerigar.All())
	h.budgerigar.Clear()

	return map[string]any{"deletedCount": deletedCount}, nil
}

func mcpStubsSearch(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return nil, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	if method == "" {
		return nil, mcpRequiredArgError("method")
	}

	input, err := mcpSearchInput(args)
	if err != nil {
		return nil, err
	}

	headers, err := mcpHeadersArg(args)
	if err != nil {
		return nil, err
	}

	sessionID, _ := args["session"].(string)

	result, searchErr := h.budgerigar.FindByQuery(stuber.Query{
		Service: service,
		Method:  method,
		Session: sessionID,
		Headers: headers,
		Input:   input,
	})
	if searchErr != nil {
		return mcpSearchNotMatchedResponse(searchErr), nil
	}

	found := result.Found()
	if found == nil {
		response := map[string]any{"matched": false}

		if similar := result.Similar(); similar != nil {
			response["similarStubId"] = similar.ID.String()
		}

		return response, nil
	}

	return map[string]any{
		"matched": true,
		"stubId":  found.ID.String(),
		"output":  found.Output,
	}, nil
}

func mcpStubsInspect(h *RestServer, args map[string]any) (map[string]any, error) {
	query, err := mcpInspectQuery(args)
	if err != nil {
		return nil, err
	}

	report := h.budgerigar.InspectQuery(query)

	return map[string]any{"report": toRestInspectReport(report)}, nil
}

func mcpInspectQuery(args map[string]any) (stuber.Query, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return stuber.Query{}, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	if method == "" {
		return stuber.Query{}, mcpRequiredArgError("method")
	}

	query := stuber.Query{Service: service, Method: method}

	err := mcpInspectQueryOptions(args, &query)
	if err != nil {
		return stuber.Query{}, err
	}

	return query, nil
}

func mcpInspectQueryOptions(args map[string]any, query *stuber.Query) error {
	if query == nil {
		return nil
	}

	if idValue, ok := args["id"]; ok && idValue != nil {
		id, err := mcpUUIDArg(args, "id")
		if err != nil {
			return err
		}

		query.ID = &id
	}

	if sessionID, _ := args["session"].(string); sessionID != "" {
		query.Session = sessionID
	}

	headers, err := mcpHeadersArg(args)
	if err != nil {
		return err
	}

	query.Headers = headers

	if rawInput, ok := args["input"]; ok && rawInput != nil {
		input, err := parseMCPInputArg(rawInput)
		if err != nil {
			return err
		}

		query.Input = input
	}

	return nil
}

func decodeMCPStubsArg(raw any) ([]*stuber.Stub, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, mcpStubPayloadArgError(err)
	}

	var items []*stuber.Stub
	if err = jsondecoder.UnmarshalSlice(payload, &items); err != nil {
		return nil, mcpStubPayloadArgError(err)
	}

	if len(items) == 0 {
		return nil, mcpInvalidArgError("stubs cannot be empty")
	}

	return items, nil
}

func listMCPStubs(stubs []*stuber.Stub, args map[string]any) ([]*stuber.Stub, error) {
	service, _ := args["service"].(string)
	method, _ := args["method"].(string)
	sessionID, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	offset, err := mcpIntArg(args, "offset", 0)
	if err != nil {
		return nil, err
	}

	filtered := filterMCPStubs(stubs, service, method, sessionID)

	if offset >= len(filtered) {
		return []*stuber.Stub{}, nil
	}

	filtered = filtered[offset:]

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func mcpUUIDArg(args map[string]any, key string) (uuid.UUID, error) {
	value, _ := args[key].(string)
	if value == "" {
		return uuid.Nil, mcpRequiredArgError(key)
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, mcpUUIDArgError(key, value, err)
	}

	return id, nil
}

func mcpStringSliceArg(args map[string]any, key string) ([]string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, mcpRequiredArgError(key)
	}

	switch values := raw.(type) {
	case []string:
		return validateMCPStringSlice(values, key)
	case []any:
		return convertMCPAnyStringSlice(values, key)
	default:
		return nil, mcpStringListArgError(key)
	}
}

func mcpSearchInput(args map[string]any) ([]map[string]any, error) {
	if rawInput, ok := args["input"]; ok && rawInput != nil {
		return parseMCPInputArg(rawInput)
	}

	payload, ok := args["payload"].(map[string]any)
	if !ok || payload == nil {
		return nil, mcpRequiredArgError("payload")
	}

	return []map[string]any{payload}, nil
}

func mcpHeadersArg(args map[string]any) (map[string]any, error) {
	rawHeaders, ok := args["headers"]
	if !ok || rawHeaders == nil {
		return map[string]any{}, nil
	}

	headers, ok := rawHeaders.(map[string]any)
	if !ok {
		return nil, mcpInvalidArgError("headers must be an object")
	}

	return headers, nil
}

func mcpSearchNotMatchedResponse(searchErr error) map[string]any {
	return map[string]any{"matched": false, "error": searchErr.Error()}
}

func filterMCPStubs(stubs []*stuber.Stub, service, method, sessionID string) []*stuber.Stub {
	filtered := make([]*stuber.Stub, 0, len(stubs))

	for _, stub := range stubs {
		if !mcpStubMatchesFilters(stub, service, method, sessionID) {
			continue
		}

		filtered = append(filtered, stub)
	}

	return filtered
}

func mcpStubMatchesFilters(stub *stuber.Stub, service, method, sessionID string) bool {
	if service != "" && stub.Service != service {
		return false
	}

	if method != "" && stub.Method != method {
		return false
	}

	return stubVisibleForSession(stub.Session, sessionID)
}

func validateMCPStringSlice(values []string, key string) ([]string, error) {
	if len(values) == 0 {
		return nil, mcpInvalidArgError(key + " cannot be empty")
	}

	if slices.Contains(values, "") {
		return nil, mcpStringListArgError(key)
	}

	return values, nil
}

func convertMCPAnyStringSlice(values []any, key string) ([]string, error) {
	if len(values) == 0 {
		return nil, mcpInvalidArgError(key + " cannot be empty")
	}

	out := make([]string, 0, len(values))
	for _, item := range values {
		value, ok := item.(string)
		if !ok || value == "" {
			return nil, mcpStringListArgError(key)
		}

		out = append(out, value)
	}

	return out, nil
}

func parseMCPInputArg(rawInput any) ([]map[string]any, error) {
	switch input := rawInput.(type) {
	case []map[string]any:
		if len(input) == 0 {
			return nil, mcpInvalidArgError("input cannot be empty")
		}

		return input, nil
	case []any:
		if len(input) == 0 {
			return nil, mcpInvalidArgError("input cannot be empty")
		}

		return convertMCPAnyMapSlice(input)
	default:
		return nil, mcpInvalidArgError("input must be an array")
	}
}

func convertMCPAnyMapSlice(input []any) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(input))
	for _, item := range input {
		message, ok := item.(map[string]any)
		if !ok {
			return nil, mcpInvalidArgError("input must contain JSON objects")
		}

		out = append(out, message)
	}

	return out, nil
}

func uuidListToStringSlice(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))

	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}

func debugCall(h *RestServer, service, method, session string, historyLimit, stubsLimit int) map[string]any {
	serviceFound, methodFound := lookupServiceAndMethod(h, service, method)
	dynamic := slices.Contains(h.restDescriptors.ServiceIDs(), service)
	stubCount, stubRecords := collectDebugStubs(h, service, method, session, stubsLimit)
	historyRecords := filterHistory(h, history.FilterOpts{Service: service, Method: method, Session: session}, historyLimit)
	errorRecords := extractErrorRecords(historyRecords)
	hints := buildDebugHints(h, serviceFound, methodFound, method, stubCount)

	return map[string]any{
		"service":           service,
		"method":            method,
		"session":           session,
		"serviceRegistered": serviceFound,
		"methodRegistered":  methodFound,
		"dynamicService":    dynamic,
		"stubCount":         stubCount,
		"stubs":             stubRecords,
		"historyCount":      len(historyRecords),
		"errorCount":        len(errorRecords),
		"recentHistory":     historyRecords,
		"recentErrors":      errorRecords,
		"hints":             hints,
	}
}

func lookupServiceAndMethod(h *RestServer, service, method string) (bool, bool) {
	for _, svc := range h.collectAllServices() {
		if svc.Id != service {
			continue
		}

		if method == "" {
			return true, true
		}

		for _, m := range svc.Methods {
			if m.Name == method || m.Id == service+"/"+method {
				return true, true
			}
		}

		return true, false
	}

	return false, false
}

func collectDebugStubs(h *RestServer, service, method, session string, stubsLimit int) (int, []map[string]any) {
	stubRecords := make([]map[string]any, 0)
	stubCount := 0

	for _, stub := range h.budgerigar.All() {
		if stub.Service != service {
			continue
		}

		if !stubVisibleForSession(stub.Session, session) {
			continue
		}

		if method != "" && stub.Method != method {
			continue
		}

		stubCount++

		if stubsLimit > 0 && len(stubRecords) >= stubsLimit {
			continue
		}

		stubRecords = append(stubRecords, map[string]any{
			"id":       stub.ID.String(),
			"service":  stub.Service,
			"method":   stub.Method,
			"session":  stub.Session,
			"priority": stub.Priority,
		})
	}

	return stubCount, stubRecords
}

func stubVisibleForSession(stubSession, querySession string) bool {
	if querySession == "" {
		return stubSession == ""
	}

	return stubSession == "" || stubSession == querySession
}

func extractErrorRecords(records []rest.CallRecord) []rest.CallRecord {
	errorsOnly := make([]rest.CallRecord, 0)

	for _, item := range records {
		if item.Error != nil && *item.Error != "" {
			errorsOnly = append(errorsOnly, item)
		}
	}

	return errorsOnly
}

func buildDebugHints(h *RestServer, serviceFound, methodFound bool, method string, stubCount int) []string {
	hints := make([]string, 0, debugCallHintsCap)

	if !serviceFound {
		hints = append(hints, "Service is not registered. Add descriptors first (MCP descriptors.add).")
	}

	if serviceFound && method != "" && !methodFound {
		hints = append(hints, "Method is not found in service descriptor.")
	}

	if serviceFound && methodFound && stubCount == 0 {
		hints = append(hints, "No stubs found for this service/method. Add one via /api/stubs.")
	}

	if h.history == nil {
		hints = append(hints, "History is disabled; enable HISTORY_ENABLED=true to inspect call traces.")
	}

	return hints
}

func filterHistory(h *RestServer, opts history.FilterOpts, limit int) []rest.CallRecord {
	if h.history == nil {
		return []rest.CallRecord{}
	}

	calls := h.history.Filter(opts)
	if limit > 0 && len(calls) > limit {
		calls = calls[len(calls)-limit:]
	}

	out := make([]rest.CallRecord, len(calls))
	for i, c := range calls {
		out[i] = historyCallRecordToRest(c)
	}

	return out
}

func mcpIntArg(args map[string]any, key string, defaultValue int) (int, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return defaultValue, nil
	}

	switch v := raw.(type) {
	case float64:
		if v < 0 || v != float64(int(v)) {
			return 0, mcpNonNegativeIntegerArgError(key)
		}

		return int(v), nil
	case int:
		if v < 0 {
			return 0, mcpNonNegativeIntegerArgError(key)
		}

		return v, nil
	default:
		return 0, mcpNonNegativeIntegerArgError(key)
	}
}

// ListHistory returns recorded gRPC calls.
func (h *RestServer) ListHistory(w http.ResponseWriter, r *http.Request) {
	if h.history == nil {
		h.writeResponse(r.Context(), w, rest.HistoryList{})

		return
	}

	calls := h.history.Filter(history.FilterOpts{Session: muxmiddleware.FromRequest(r)})

	out := make(rest.HistoryList, len(calls))
	for i, c := range calls {
		out[i] = historyCallRecordToRest(c)
	}

	h.writeResponse(r.Context(), w, out)
}

func historyCallRecordToRest(c history.CallRecord) rest.CallRecord {
	r := rest.CallRecord{
		Service: new(c.Service),
		Method:  new(c.Method),
	}

	if c.StubID != uuid.Nil {
		r.StubId = &c.StubID
	}

	if len(c.Requests) > 0 {
		r.Requests = &c.Requests
		r.Request = &c.Requests[0]
	} else if c.Request != nil {
		r.Request = &c.Request
	}

	if len(c.Responses) > 0 {
		r.Responses = &c.Responses
		r.Response = &c.Responses[0]
	} else if c.Response != nil {
		r.Response = &c.Response
	}

	if c.Error != "" {
		r.Error = &c.Error
	}

	if c.Code != 0 {
		code := int(c.Code)
		r.Code = &code
	}

	if !c.Timestamp.IsZero() {
		r.Timestamp = &c.Timestamp
	}

	return r
}

// VerifyCalls verifies that a method was called the expected number of times.
func (h *RestServer) VerifyCalls(w http.ResponseWriter, r *http.Request) {
	if h.history == nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponse(r.Context(), w, rest.VerifyError{Message: new("history is disabled")})

		return
	}

	var req rest.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, errors.Wrap(err, "invalid verify request"))

		return
	}

	calls := h.history.Filter(history.FilterOpts{
		Service: req.Service,
		Method:  req.Method,
		Session: muxmiddleware.FromRequest(r),
	})

	actual := len(calls)
	if actual != req.ExpectedCount {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponse(r.Context(), w, rest.VerifyError{
			Message:  new(fmt.Sprintf("expected %s/%s to be called %d times, got %d", req.Service, req.Method, req.ExpectedCount, actual)),
			Expected: &req.ExpectedCount,
			Actual:   &actual,
		})

		return
	}

	h.writeResponse(r.Context(), w, rest.MessageOK{Message: "ok", Time: time.Now()})
}

// AddStub inserts new stubs.
func (h *RestServer) AddStub(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var inputs []*stuber.Stub

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	sess := muxmiddleware.FromRequest(r)
	for _, stub := range inputs {
		stub.Session = sess
		stub.Source = stuber.SourceRest

		if err := h.validateStub(stub); err != nil {
			h.validationError(r.Context(), w, err)

			return
		}
	}

	h.writeResponse(r.Context(), w, h.budgerigar.PutMany(inputs...))
}

// ListDescriptors returns service IDs of descriptors added via POST /descriptors.
func (h *RestServer) ListDescriptors(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, rest.DescriptorServiceIDs{ServiceIDs: h.restDescriptors.ServiceIDs()})
}

// AddDescriptors accepts binary FileDescriptorSet and registers it for discovery.
// Returns service IDs; use DELETE /services/{serviceID} to remove.
func (h *RestServer) AddDescriptors(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	if len(byt) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, ErrEmptyBody)

		return
	}

	serviceIDs, err := registerDescriptorBytes(h, byt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, err)

		return
	}

	h.writeResponse(r.Context(), w, rest.AddDescriptorsResponse{
		Message:    "ok",
		Time:       time.Now(),
		ServiceIDs: serviceIDs,
	})
}

// DeleteService removes a service added via POST /descriptors.
// Services from startup (proto path) cannot be removed and return 404.
func (h *RestServer) DeleteService(w http.ResponseWriter, _ *http.Request, serviceID string) {
	if unregisterService(h, serviceID) == 0 {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(context.Background(), w, serviceNotRemovable(serviceID))

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func unregisterService(h *RestServer, serviceID string) int {
	h.descriptorOpsMu.Lock()
	defer h.descriptorOpsMu.Unlock()

	return h.restDescriptors.UnregisterByService(serviceID)
}

func registerDescriptorBytes(h *RestServer, byt []byte) ([]string, error) {
	h.descriptorOpsMu.Lock()
	defer h.descriptorOpsMu.Unlock()

	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(byt, &fds); err != nil {
		return nil, invalidFileDescriptorSetError(err)
	}

	if len(fds.GetFile()) == 0 {
		return nil, ErrFileDescriptorSetNoFiles
	}

	files, err := decodeDescriptorFiles(&fds)
	if err != nil {
		return nil, err
	}

	serviceIDs := make([]string, 0)

	for _, fd := range files {
		h.restDescriptors.Register(fd)

		services := fd.Services()
		for i := range services.Len() {
			serviceIDs = append(serviceIDs, string(services.Get(i).FullName()))
		}
	}

	sort.Strings(serviceIDs)

	return serviceIDs, nil
}

func decodeDescriptorFiles(fds *descriptorpb.FileDescriptorSet) ([]protoreflect.FileDescriptor, error) {
	registry := new(protoregistry.Files)
	pending := make([]*descriptorpb.FileDescriptorProto, 0, len(fds.GetFile()))

	for _, fd := range fds.GetFile() {
		if fd != nil {
			pending = append(pending, fd)
		}
	}

	for len(pending) > 0 {
		progress := false
		nextPending := make([]*descriptorpb.FileDescriptorProto, 0, len(pending))

		resolver := &protosetinfra.Fallback{Primary: registry, Fallback: protoregistry.GlobalFiles}

		for _, fd := range pending {
			fileDesc, err := protodesc.NewFile(fd, resolver)
			if err != nil {
				nextPending = append(nextPending, fd)

				continue
			}

			if err := registry.RegisterFile(fileDesc); err != nil {
				return nil, registerDescriptorFileError(fd.GetName(), err)
			}

			progress = true
		}

		if !progress {
			return nil, ErrResolveDescriptorDeps
		}

		pending = nextPending
	}

	files := make([]protoreflect.FileDescriptor, 0, len(fds.GetFile()))

	registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		files = append(files, fd)

		return true
	})

	return files, nil
}

// DeleteStubByID removes a stub by ID.
func (h *RestServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	h.budgerigar.DeleteByID(uuid)

	w.WriteHeader(http.StatusNoContent)
}

// BatchStubsDelete removes multiple stubs by ID.
func (h *RestServer) BatchStubsDelete(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var inputs []uuid.UUID

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	if len(inputs) > 0 {
		h.budgerigar.DeleteByID(inputs...)
	}
}

// ListUsedStubs returns stubs that have been matched.
func (h *RestServer) ListUsedStubs(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.budgerigar.Used())
}

// ListUnusedStubs returns stubs that have never been matched.
func (h *RestServer) ListUnusedStubs(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.budgerigar.Unused())
}

// ListStubs returns all stubs, optionally filtered by source.
func (h *RestServer) ListStubs(w http.ResponseWriter, r *http.Request, params rest.ListStubsParams) {
	stubs, total := h.budgerigar.List(listOptionsFromParams(params))
	w.Header().Set("X-Total-Count", strconv.Itoa(total))

	h.writeResponse(r.Context(), w, stubs)
}

func listOptionsFromParams(params rest.ListStubsParams) stuber.ListOptions {
	options := stuber.ListOptions{
		Source:  stringFromPtr(params.Source),
		Service: stringFromPtr(params.Service),
		Method:  stringFromPtr(params.Method),
		Sort:    stringFromPtr(params.Sort),
		Limit:   intFromPtr(params.Limit),
		Offset:  intFromPtr(params.Offset),
	}

	if params.Session != nil {
		options.Session = *params.Session
		options.SessionSet = true
	}

	return options
}

// PurgeStubs removes all stubs.
func (h *RestServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.budgerigar.Clear()

	w.WriteHeader(http.StatusNoContent)
}

// SearchStubs finds a stub matching the query.
func (h *RestServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	query, err := stuber.NewQuery(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	defer func() {
		_ = r.Body.Close()
	}()

	if sess := muxmiddleware.FromRequest(r); sess != "" {
		query.Session = sess
	}

	result, err := h.budgerigar.FindByQuery(query)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, err)

		return
	}

	if result.Found() == nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, stubNotFoundError(query, result))

		return
	}

	h.writeResponse(r.Context(), w, result.Found().Output)
}

// InspectStubs returns detailed matching report for a query.
func (h *RestServer) InspectStubs(w http.ResponseWriter, r *http.Request) {
	var req rest.InspectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	query := stuber.Query{
		Service: req.Service,
		Method:  req.Method,
		Input:   req.Input,
		Headers: req.Headers,
	}

	if req.Id != nil {
		id := *req.Id
		query.ID = &id
	}

	if req.Session != nil {
		query.Session = *req.Session
	}

	report := h.budgerigar.InspectQuery(query)
	h.writeResponse(r.Context(), w, toRestInspectReport(report))
}

func toRestInspectReport(report stuber.InspectReport) rest.InspectReport {
	stages := make([]rest.InspectStage, len(report.Stages))
	for i, stage := range report.Stages {
		stages[i] = rest.InspectStage{
			Name:    stage.Name,
			Before:  stage.Before,
			After:   stage.After,
			Removed: stage.Removed,
		}
	}

	candidates := make([]rest.InspectCandidate, len(report.Candidates))
	for i, candidate := range report.Candidates {
		events := make([]rest.InspectCandidateEvent, len(candidate.Events))
		for j, event := range candidate.Events {
			reason := event.Reason
			events[j] = rest.InspectCandidateEvent{
				Stage:  event.Stage,
				Result: event.Result,
				Reason: nilIfEmpty(reason),
			}
		}

		candidates[i] = rest.InspectCandidate{
			Id:               candidate.ID.String(),
			Service:          candidate.Service,
			Method:           candidate.Method,
			Session:          candidate.Session,
			Priority:         candidate.Priority,
			Times:            candidate.Times,
			Used:             candidate.Used,
			Specificity:      candidate.Specificity,
			Score:            candidate.Score,
			VisibleBySession: candidate.VisibleBySession,
			WithinTimes:      candidate.WithinTimes,
			HeadersMatched:   candidate.HeadersMatched,
			InputMatched:     candidate.InputMatched,
			Matched:          candidate.Matched,
			ExcludedBy:       candidate.ExcludedBy,
			Events:           events,
		}
	}

	return rest.InspectReport{
		Service:          report.Service,
		Method:           report.Method,
		Session:          report.Session,
		MatchedStubId:    stringFromUUIDPtr(report.MatchedStubID),
		SimilarStubId:    stringFromUUIDPtr(report.SimilarStubID),
		FallbackToMethod: report.FallbackToMethod,
		Error:            stringFromPtr(report.Error),
		Stages:           stages,
		Candidates:       candidates,
	}
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func stringFromPtr(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func intFromPtr(value *int) int {
	if value == nil {
		return 0
	}

	return *value
}

func stringFromUUIDPtr(value *uuid.UUID) string {
	if value == nil {
		return ""
	}

	return value.String()
}

func (h *RestServer) collectServices(file protoreflect.FileDescriptor, results *[]rest.Service) bool {
	services := file.Services()

	for i := range services.Len() {
		*results = append(*results, h.serviceFromDescriptor(services.Get(i), false))
	}

	return true
}

func (h *RestServer) collectAllServices() []rest.Service {
	results := make([]rest.Service, 0, servicesListCap)

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return h.collectServices(file, &results)
	})

	h.restDescriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return h.collectServices(file, &results)
	})

	sort.Slice(results, func(i, j int) bool {
		return results[i].Id < results[j].Id
	})

	return results
}

func (h *RestServer) serviceFromDescriptor(
	service protoreflect.ServiceDescriptor,
	includeSchemas bool,
) rest.Service {
	methods := service.Methods()
	result := rest.Service{
		Id:      string(service.FullName()),
		Name:    string(service.Name()),
		Package: string(service.ParentFile().Package()),
		Methods: make([]rest.Method, 0, methods.Len()),
	}

	for j := range methods.Len() {
		result.Methods = append(result.Methods, h.methodFromDescriptor(service, methods.Get(j), includeSchemas))
	}

	sort.Slice(result.Methods, func(i, j int) bool {
		return result.Methods[i].Id < result.Methods[j].Id
	})

	return result
}

func (h *RestServer) methodFromDescriptor(
	service protoreflect.ServiceDescriptor,
	method protoreflect.MethodDescriptor,
	includeSchemas bool,
) rest.Method {
	requestType := string(method.Input().FullName())
	responseType := string(method.Output().FullName())

	result := rest.Method{
		Id:              fmt.Sprintf("%s/%s", string(service.FullName()), string(method.Name())),
		Name:            string(method.Name()),
		MethodType:      grpcMethodType(method.IsStreamingClient(), method.IsStreamingServer()),
		RequestType:     &requestType,
		ResponseType:    &responseType,
		ClientStreaming: method.IsStreamingClient(),
		ServerStreaming: method.IsStreamingServer(),
	}

	if includeSchemas {
		result.RequestSchema = h.messageSchemaFromDescriptor(method.Input(), map[protoreflect.FullName]struct{}{})
		result.ResponseSchema = h.messageSchemaFromDescriptor(method.Output(), map[protoreflect.FullName]struct{}{})
	}

	return result
}

func (h *RestServer) messageSchemaFromDescriptor(
	message protoreflect.MessageDescriptor,
	visiting map[protoreflect.FullName]struct{},
) *rest.ProtoMessageSchema {
	fullName := message.FullName()
	if _, ok := visiting[fullName]; ok {
		return &rest.ProtoMessageSchema{
			TypeName:     string(fullName),
			Fields:       []rest.ProtoFieldSchema{},
			RecursiveRef: true,
		}
	}

	visiting[fullName] = struct{}{}
	defer delete(visiting, fullName)

	fields := message.Fields()
	result := rest.ProtoMessageSchema{
		TypeName: string(fullName),
		Fields:   make([]rest.ProtoFieldSchema, 0, fields.Len()),
	}

	for i := range fields.Len() {
		result.Fields = append(result.Fields, h.fieldSchemaFromDescriptor(fields.Get(i), visiting))
	}

	return &result
}

//nolint:funlen
func (h *RestServer) fieldSchemaFromDescriptor(
	field protoreflect.FieldDescriptor,
	visiting map[protoreflect.FullName]struct{},
) rest.ProtoFieldSchema {
	result := rest.ProtoFieldSchema{
		Name:        string(field.Name()),
		JsonName:    field.JSONName(),
		Number:      int(field.Number()),
		Kind:        field.Kind().String(),
		Cardinality: grpcCardinality(field.Cardinality()),
	}

	if oneof := field.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() {
		group := string(oneof.Name())
		result.Oneof = &group
	}

	if field.IsMap() {
		result.Map = true

		keyKind := field.MapKey().Kind().String()
		result.MapKeyKind = &keyKind

		mapValue := field.MapValue()
		valueKind := mapValue.Kind().String()
		result.MapValueKind = &valueKind

		if mapValue.Kind() == protoreflect.MessageKind {
			valueTypeName := string(mapValue.Message().FullName())
			result.MapValueTypeName = &valueTypeName
		}

		if mapValue.Kind() == protoreflect.EnumKind {
			valueTypeName := string(mapValue.Enum().FullName())
			result.MapValueTypeName = &valueTypeName
		}

		if mapValue.Kind() == protoreflect.MessageKind {
			result.MapValueMessage = h.messageSchemaFromDescriptor(mapValue.Message(), visiting)
		}

		return result
	}

	if field.Kind() == protoreflect.EnumKind {
		enumTypeName := string(field.Enum().FullName())
		result.TypeName = &enumTypeName

		enumValues := make([]string, 0, field.Enum().Values().Len())
		for i := range field.Enum().Values().Len() {
			enumValues = append(enumValues, string(field.Enum().Values().Get(i).Name()))
		}

		result.EnumValues = &enumValues

		return result
	}

	if field.Kind() == protoreflect.MessageKind {
		messageTypeName := string(field.Message().FullName())
		result.TypeName = &messageTypeName
		result.Message = h.messageSchemaFromDescriptor(field.Message(), visiting)
	}

	return result
}

func grpcCardinality(cardinality protoreflect.Cardinality) rest.ProtoFieldSchemaCardinality {
	switch cardinality {
	case protoreflect.Required:
		return rest.Required
	case protoreflect.Repeated:
		return rest.Repeated
	case protoreflect.Optional:
		return rest.Optional
	default:
		return rest.Optional
	}
}

func grpcMethodType(clientStreaming bool, serverStreaming bool) rest.MethodMethodType {
	switch {
	case clientStreaming && serverStreaming:
		return rest.BidiStreaming
	case clientStreaming:
		return rest.ClientStreaming
	case serverStreaming:
		return rest.ServerStreaming
	default:
		return rest.Unary
	}
}

// liveness handles the liveness probe response.
func (h *RestServer) liveness(ctx context.Context, w http.ResponseWriter) {
	h.writeResponse(ctx, w, rest.MessageOK{Message: "ok", Time: time.Now()})
}

// responseError writes an error response to the HTTP writer.
func (h *RestServer) responseError(ctx context.Context, w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(ctx, w, err)
}

// validationError writes a validation error response to the HTTP writer.
func (h *RestServer) validationError(ctx context.Context, w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)

	h.writeResponseError(ctx, w, err)
}

// writeResponseError writes an error response to the HTTP writer.
func (h *RestServer) writeResponseError(ctx context.Context, w http.ResponseWriter, err error) {
	h.writeResponse(ctx, w, map[string]string{
		"error": err.Error(),
	})
}

// writeResponse writes a successful response to the HTTP writer.
func (h *RestServer) writeResponse(ctx context.Context, w http.ResponseWriter, data any) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to encode JSON response")
	}
}

// validateStub validates if the stub is valid or not.
func (h *RestServer) validateStub(stub *stuber.Stub) error {
	if err := h.validator.Struct(stub); err != nil {
		validationErrors, ok := stderrors.AsType[validator.ValidationErrors](err)
		if !ok {
			return err
		}

		if len(validationErrors) > 0 {
			fieldError := validationErrors[0]

			return &ValidationError{
				Field:   fieldError.Field(),
				Tag:     fieldError.Tag(),
				Value:   fieldError.Value(),
				Message: getValidationMessage(fieldError),
			}
		}

		return err
	}

	return nil
}

func (h *RestServer) dashboardPayload(r *http.Request) rest.Dashboard {
	all := h.budgerigar.All()
	used := h.budgerigar.Used()

	payload := rest.Dashboard{
		AppName:            "gripmock",
		Version:            build.Version,
		GoVersion:          runtime.Version(),
		Compiler:           runtime.Compiler,
		Goos:               runtime.GOOS,
		Goarch:             runtime.GOARCH,
		NumCPU:             runtime.NumCPU(),
		StartedAt:          h.startedAt,
		UptimeSeconds:      int(time.Since(h.startedAt).Seconds()),
		Ready:              h.ok.Load(),
		HistoryEnabled:     h.history != nil,
		TotalServices:      len(h.collectAllServices()),
		TotalStubs:         len(all),
		UsedStubs:          len(used),
		UnusedStubs:        max(len(all)-len(used), 0),
		TotalSessions:      len(h.budgerigar.Sessions()),
		RuntimeDescriptors: len(h.restDescriptors.ServiceIDs()),
		TotalHistory:       0,
		HistoryErrors:      0,
	}

	if h.history == nil {
		return payload
	}

	records := h.history.Filter(history.FilterOpts{Session: muxmiddleware.FromRequest(r)})
	payload.TotalHistory = len(records)

	for _, record := range records {
		if record.Error != "" {
			payload.HistoryErrors++
		}
	}

	return payload
}

func (h *RestServer) findServiceDetailed(serviceID string) (rest.Service, bool) {
	serviceDescriptor, ok := h.findServiceDescriptor(serviceID)
	if !ok {
		return rest.Service{}, false
	}

	return h.serviceFromDescriptor(serviceDescriptor, true), true
}

func (h *RestServer) findServiceDescriptor(serviceID string) (protoreflect.ServiceDescriptor, bool) { //nolint:ireturn
	var found protoreflect.ServiceDescriptor

	collect := func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			service := services.Get(i)
			if string(service.FullName()) == serviceID {
				found = service

				return false
			}
		}

		return true
	}

	if strings.Contains(serviceID, ".") {
		packageName := splitLast(serviceID, ".")[0]

		protoregistry.GlobalFiles.RangeFilesByPackage(protoreflect.FullName(packageName), collect)

		if found != nil {
			return found, true
		}

		h.restDescriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
			if string(file.Package()) != packageName {
				return true
			}

			return collect(file)
		})

		if found != nil {
			return found, true
		}
	}

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return collect(file)
	})

	if found != nil {
		return found, true
	}

	h.restDescriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return collect(file)
	})

	if found == nil {
		return nil, false
	}

	return found, true
}
