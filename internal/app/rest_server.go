package app

import (
	"context"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
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
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

// Extender defines the interface for extending stub functionality.
type Extender interface {
	Wait(ctx context.Context)
}

// RestServer handles HTTP REST API requests for stub management.
type RestServer struct {
	ok              atomic.Bool
	descriptorOpsMu sync.Mutex
	budgerigar      *stuber.Budgerigar
	history         history.Reader
	validator       *validator.Validate
	restDescriptors *descriptors.Registry
}

var _ rest.ServerInterface = &RestServer{}

// NewRestServer creates a new REST server instance with the specified dependencies.
// If historyReader is nil, /api/history and /api/verify return empty/error.
// If stubValidator is nil, a shared default validator is used.
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
		v = defaultStubValidator()
	}

	r := registry
	if r == nil {
		r = descriptors.NewRegistry()
	}

	server := &RestServer{
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
	results := make([]rest.Method, 0, serviceMethodsCap)
	packageName := splitLast(serviceID, ".")[0]
	collect := h.collectMethods(serviceID, &results)

	protoregistry.GlobalFiles.RangeFilesByPackage(protoreflect.FullName(packageName), collect)
	h.restDescriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if string(file.Package()) != packageName {
			return true
		}

		return collect(file)
	})

	sort.Slice(results, func(i, j int) bool {
		return results[i].Id < results[j].Id
	})

	h.writeResponse(r.Context(), w, results)
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

const (
	debugCallDefaultLimit = 20
	debugCallHintsCap     = 4
)

func (h *RestServer) McpInfo(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, rest.McpInfoResponse{
		ProtocolVersion: mcpusecase.ProtocolVersion,
		ServerName:      "gripmock",
		ServerVersion:   build.Version,
		Methods: []string{
			"initialize",
			"ping",
			"tools/list",
			"tools/call",
		},
		Transport: rest.McpTransport{
			Path:    "/api/mcp",
			Methods: []string{http.MethodGet, http.MethodPost},
		},
	})
}

func (h *RestServer) McpMessage(w http.ResponseWriter, r *http.Request) {
	body, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponse(r.Context(), w, mcpusecase.ParsePayloadErrorResponse("invalid JSON payload"))

		return
	}

	_, hasID := raw["id"]

	var req rest.McpRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponse(r.Context(), w, mcpusecase.ParsePayloadErrorResponse("invalid JSON payload"))

		return
	}

	if req.Jsonrpc != mcpusecase.JSONRPCVersion || req.Method == "" {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponse(r.Context(), w, mcpusecase.ErrorResponse(req.Id, mcpusecase.ErrorCodeInvalidReq, mcpInvalidRequestError().Error(), nil))

		return
	}

	if !hasID {
		_ = handleMCPRequest(h, r, req)

		w.WriteHeader(http.StatusNoContent)

		return
	}

	resp := handleMCPRequest(h, r, req)
	h.writeResponse(r.Context(), w, resp)
}

func handleMCPRequest(h *RestServer, r *http.Request, req rest.McpRequest) rest.McpResponse {
	switch req.Method {
	case "initialize":
		return mcpusecase.InitializeResponse(req.Id, build.Version)
	case "ping":
		return mcpusecase.PingResponse(req.Id)
	case "tools/list":
		return mcpusecase.ToolsListResponse(req.Id, mcpusecase.ListTools())
	case "tools/call":
		return handleMCPToolCall(h, r, req)
	default:
		return mcpusecase.ErrorResponse(
			req.Id,
			mcpusecase.ErrorCodeNotFound,
			mcpRPCMethodNotFoundError().Error(),
			map[string]any{"method": req.Method},
		)
	}
}

func handleMCPToolCall(h *RestServer, r *http.Request, req rest.McpRequest) rest.McpResponse {
	toolName, _ := req.Params["name"].(string)
	if toolName == "" {
		return mcpusecase.ErrorResponse(req.Id, mcpusecase.ErrorCodeInvalidArg, mcpRequiredArgError("name").Error(), nil)
	}

	args, _ := req.Params["arguments"].(map[string]any)
	args = mcpusecase.ApplyTransportSession(r, toolName, args)

	result, err := callMCPToolDispatch(h, toolName, args)
	if err != nil {
		if stderrors.Is(err, ErrMCPInvalidArgument) {
			return mcpusecase.ErrorResponse(req.Id, mcpusecase.ErrorCodeInvalidArg, err.Error(), map[string]any{"tool": toolName})
		}

		if stderrors.Is(err, ErrMCPToolNotFound) {
			return mcpusecase.ErrorResponse(req.Id, mcpusecase.ErrorCodeNotFound, err.Error(), map[string]any{"tool": toolName})
		}

		return mcpusecase.ErrorResponse(req.Id, mcpusecase.ErrorCodeInternal, err.Error(), map[string]any{"tool": toolName})
	}

	return mcpusecase.ToolCallSuccessResponse(req.Id, result)
}

func callMCPToolDispatch(h *RestServer, name string, args map[string]any) (map[string]any, error) {
	handlers := map[string]mcpusecase.ToolHandler{
		mcpusecase.ToolDescriptorsAdd:  func(toolArgs map[string]any) (map[string]any, error) { return mcpDescriptorsAdd(h, toolArgs) },
		mcpusecase.ToolDescriptorsList: func(toolArgs map[string]any) (map[string]any, error) { return mcpDescriptorsList(h, toolArgs) },
		mcpusecase.ToolServicesList:    func(toolArgs map[string]any) (map[string]any, error) { return mcpServicesList(h, toolArgs) },
		mcpusecase.ToolServicesDelete:  func(toolArgs map[string]any) (map[string]any, error) { return mcpServicesDelete(h, toolArgs) },
		mcpusecase.ToolHistoryList:     func(toolArgs map[string]any) (map[string]any, error) { return mcpHistoryList(h, toolArgs) },
		mcpusecase.ToolHistoryErrors:   func(toolArgs map[string]any) (map[string]any, error) { return mcpHistoryErrors(h, toolArgs) },
		mcpusecase.ToolDebugCall:       func(toolArgs map[string]any) (map[string]any, error) { return mcpDebugCall(h, toolArgs) },
	}

	result, err, found := mcpusecase.DispatchTool(name, args, handlers)
	if !found {
		return nil, mcpUnknownTool(name)
	}

	return result, err
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

	calls := h.history.Filter(history.FilterOpts{Session: session.FromRequest(r)})

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
		StubId:  new(c.StubID),
	}
	if c.Request != nil {
		r.Request = &c.Request
	}

	if c.Response != nil {
		r.Response = &c.Response
	}

	if c.Error != "" {
		r.Error = &c.Error
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
		Session: session.FromRequest(r),
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

	sess := session.FromRequest(r)
	for _, stub := range inputs {
		stub.Session = sess

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

		resolver := &fallbackResolver{Primary: registry, Fallback: protoregistry.GlobalFiles}

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

// ListStubs returns all stubs.
func (h *RestServer) ListStubs(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.budgerigar.All())
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

	if sess := session.FromRequest(r); sess != "" {
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

func (h *RestServer) collectServices(file protoreflect.FileDescriptor, results *[]rest.Service) bool {
	services := file.Services()

	for i := range services.Len() {
		service := services.Get(i)
		methods := service.Methods()

		serviceResult := rest.Service{
			Id:      string(service.FullName()),
			Name:    string(service.Name()),
			Package: string(file.Package()),
			Methods: make([]rest.Method, 0, methods.Len()),
		}

		for j := range methods.Len() {
			method := methods.Get(j)
			serviceResult.Methods = append(serviceResult.Methods, rest.Method{
				Id:   fmt.Sprintf("%s/%s", string(service.FullName()), string(method.Name())),
				Name: string(method.Name()),
			})
		}

		sort.Slice(serviceResult.Methods, func(i, j int) bool {
			return serviceResult.Methods[i].Id < serviceResult.Methods[j].Id
		})

		*results = append(*results, serviceResult)
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

func (h *RestServer) collectMethods(serviceID string, results *[]rest.Method) func(protoreflect.FileDescriptor) bool {
	return func(file protoreflect.FileDescriptor) bool {
		services := file.Services()

		for i := range services.Len() {
			service := services.Get(i)

			if string(service.FullName()) != serviceID {
				continue
			}

			methods := service.Methods()

			for j := range methods.Len() {
				method := methods.Get(j)
				*results = append(*results, rest.Method{
					Id:   fmt.Sprintf("%s/%s", string(service.FullName()), string(method.Name())),
					Name: string(method.Name()),
				})
			}
		}

		return true
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
