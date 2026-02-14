package app

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
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

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// ErrServiceIsMissing is returned when the service name is not provided in the request.
var ErrServiceIsMissing = errors.New("service name is missing")

// ErrMethodIsMissing is returned when the method name is not provided in the request.
var ErrMethodIsMissing = errors.New("method name is missing")

// Extender defines the interface for extending stub functionality.
type Extender interface {
	Wait(ctx context.Context)
}

// RestServer handles HTTP REST API requests for stub management.
type RestServer struct {
	ok         atomic.Bool
	budgerigar *stuber.Budgerigar
	history    history.Reader
	validator  *validator.Validate
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
) (*RestServer, error) {
	v := stubValidator
	if v == nil {
		v = defaultStubValidator()
	}

	server := &RestServer{
		budgerigar: budgerigar,
		history:    historyReader,
		validator:  v,
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

// ServicesList returns a list of all available gRPC services.
func (h *RestServer) ServicesList(w http.ResponseWriter, r *http.Request) {
	results := make([]rest.Service, 0, servicesListCap)

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
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

			results = append(results, serviceResult)
		}

		return true
	})

	h.writeResponse(r.Context(), w, results)
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

	protoregistry.GlobalFiles.RangeFilesByPackage(protoreflect.FullName(packageName), func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			service := services.Get(i)

			if string(service.FullName()) != serviceID {
				continue
			}

			methods := service.Methods()

			for j := range methods.Len() {
				method := methods.Get(j)

				results = append(results, rest.Method{
					Id:   fmt.Sprintf("%s/%s", string(service.FullName()), string(method.Name())),
					Name: string(method.Name()),
				})
			}
		}

		return true
	})

	h.writeResponse(r.Context(), w, results)
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

const sessionHeader = "X-Gripmock-Session"

// ListHistory returns recorded gRPC calls.
func (h *RestServer) ListHistory(w http.ResponseWriter, r *http.Request) {
	if h.history == nil {
		h.writeResponse(r.Context(), w, rest.HistoryList{})

		return
	}

	calls := history.FilterBySession(h.history.All(), r.Header.Get(sessionHeader))

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

	calls := h.history.FilterByMethod(req.Service, req.Method)
	calls = history.FilterBySession(calls, r.Header.Get(sessionHeader))

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

	sess := r.Header.Get(sessionHeader)
	for _, stub := range inputs {
		if sess != "" {
			stub.Session = sess
		}

		if err := h.validateStub(stub); err != nil {
			h.validationError(r.Context(), w, err)

			return
		}
	}

	h.writeResponse(r.Context(), w, h.budgerigar.PutMany(inputs...))
}

// AddDescriptors accepts binary FileDescriptorSet and registers it to GlobalFiles.
func (h *RestServer) AddDescriptors(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	if len(byt) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, errors.New("empty body"))

		return
	}

	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(byt, &fds); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, errors.Wrap(err, "invalid FileDescriptorSet"))

		return
	}

	for _, fd := range fds.GetFile() {
		if fd == nil {
			continue
		}

		if _, err := protoregistry.GlobalFiles.FindFileByPath(fd.GetName()); err == nil {
			continue // already registered
		}

		fileDesc, err := protodesc.NewFile(fd, protoregistry.GlobalFiles)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			h.writeResponseError(r.Context(), w, errors.Wrapf(err, "failed to load file %s", fd.GetName()))

			return
		}

		if err := protoregistry.GlobalFiles.RegisterFile(fileDesc); err != nil {
			h.responseError(r.Context(), w, errors.Wrapf(err, "failed to register file %s", fd.GetName()))

			return
		}
	}

	h.writeResponse(r.Context(), w, rest.MessageOK{Message: "ok", Time: time.Now()})
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

// PatchStubByID partially updates a stub by ID.
func (h *RestServer) PatchStubByID(w http.ResponseWriter, r *http.Request, id rest.ID) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var patch rest.StubPatch
	if err := json.Unmarshal(byt, &patch); err != nil {
		h.validationError(r.Context(), w, err)

		return
	}

	existing := h.budgerigar.FindByID(id)
	if existing == nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponse(r.Context(), w, map[string]string{
			"error": fmt.Sprintf("Stub with ID '%s' not found", id),
		})

		return
	}

	updated := applyPatch(existing, &patch)
	if err := h.validateStub(updated); err != nil {
		h.validationError(r.Context(), w, err)

		return
	}

	h.budgerigar.UpdateMany(updated)
	h.writeResponse(r.Context(), w, updated)
}

func applyPatch(stub *stuber.Stub, patch *rest.StubPatch) *stuber.Stub {
	out := *stub

	out.Headers = applyHeadersPatch(stub.Headers, patch.Headers)
	out.Service = applyOptString(stub.Service, patch.Service)
	out.Method = applyOptString(stub.Method, patch.Method)
	out.Priority = applyOptInt(stub.Priority, patch.Priority)
	applyInputPatch(&out, patch)

	if patch.Output != nil {
		out.Output = restOutputToStuber(*patch.Output)
	}

	if patch.Options != nil {
		out.Options = stuber.StubOptions{Times: patch.Options.Times}
	}

	return &out
}

func applyHeadersPatch(cur stuber.InputHeader, p rest.StubHeaders) stuber.InputHeader {
	if p.Equals == nil && p.Contains == nil && p.Matches == nil {
		return cur
	}

	return restHeadersToStuber(p)
}

func applyOptString(cur string, p *string) string {
	if p != nil {
		return *p
	}

	return cur
}

func applyOptInt(cur int, p *int) int {
	if p != nil {
		return *p
	}

	return cur
}

func applyInputPatch(out *stuber.Stub, patch *rest.StubPatch) {
	switch {
	case patch.Input != nil:
		out.Input = restInputToStuber(*patch.Input)
		out.Inputs = nil
	case patch.Inputs != nil:
		inputs := make([]stuber.InputData, len(*patch.Inputs))
		for i, in := range *patch.Inputs {
			inputs[i] = restInputToStuber(in)
		}

		out.Inputs = inputs
		out.Input = stuber.InputData{}
	}
}

func restHeadersToStuber(h rest.StubHeaders) stuber.InputHeader {
	out := stuber.InputHeader{}
	if h.Equals != nil {
		out.Equals = make(map[string]any, len(h.Equals))
		for k, v := range h.Equals {
			out.Equals[k] = v
		}
	}

	if h.Contains != nil {
		out.Contains = make(map[string]any, len(h.Contains))
		for k, v := range h.Contains {
			out.Contains[k] = v
		}
	}

	if h.Matches != nil {
		out.Matches = make(map[string]any, len(h.Matches))
		for k, v := range h.Matches {
			out.Matches[k] = v
		}
	}

	return out
}

func restInputToStuber(in rest.StubInput) stuber.InputData {
	out := stuber.InputData{
		IgnoreArrayOrder: in.IgnoreArrayOrder,
		Equals:           in.Equals,
		Contains:         in.Contains,
		Matches:          in.Matches,
	}
	if out.Equals == nil {
		out.Equals = map[string]any{}
	}

	if out.Contains == nil {
		out.Contains = map[string]any{}
	}

	if out.Matches == nil {
		out.Matches = map[string]any{}
	}

	return out
}

func restOutputToStuber(o rest.StubOutput) stuber.Output {
	var stream []any
	if len(o.Stream) > 0 {
		stream = make([]any, len(o.Stream))
		for i, m := range o.Stream {
			stream[i] = m
		}
	}

	out := stuber.Output{
		Data:    o.Data,
		Stream:  stream,
		Headers: o.Headers,
		Error:   o.Error,
		Delay:   o.Delay,
	}
	if o.Code != 0 {
		out.Code = &o.Code
	}

	if out.Headers == nil {
		out.Headers = map[string]string{}
	}

	return out
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
