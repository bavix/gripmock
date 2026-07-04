package app

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"runtime"
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
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

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
	errorFormatter  *ErrorFormatter
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
	errorFormatter *ErrorFormatter,
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

	e := errorFormatter
	if e == nil {
		e = NewErrorFormatter()
	}

	server := &RestServer{
		startedAt:       time.Now(),
		budgerigar:      budgerigar,
		history:         historyReader,
		validator:       v,
		restDescriptors: r,
		errorFormatter:  e,
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
func (h *RestServer) DeleteService(w http.ResponseWriter, r *http.Request, serviceID string) {
	if unregisterService(h, serviceID) == 0 {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, serviceNotRemovable(serviceID))

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
		h.writeResponseError(r.Context(), w, h.errorFormatter.FormatStubNotFoundError(query, result))

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
