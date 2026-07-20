package app

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/build"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
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
	ports           ServerPorts
}

// ServerPorts holds the listen addresses of the protocol endpoints, surfaced on
// the dashboard so users know where to point clients.
type ServerPorts struct {
	GRPC    string // native gRPC
	Gateway string // ConnectRPC + gRPC-web (one port)
	HTTP    string // admin REST API + this UI
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

// SetPorts records the protocol listen addresses for the dashboard (optional).
func (h *RestServer) SetPorts(p ServerPorts) { h.ports = p }

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

// methodCoverage counts how many known gRPC methods have at least one stub.
// A stub's Service may be the FQN or the bare service name (package optional),
// so a method is covered when a stub matches either form.
func methodCoverage(services []rest.Service, stubs []*stuber.Stub) (int, int) {
	type ref struct{ service, method string }

	have := make(map[ref]struct{}, len(stubs))
	for _, s := range stubs {
		have[ref{s.Service, s.Method}] = struct{}{}
	}

	var covered, total int

	for _, svc := range services {
		for _, m := range svc.Methods {
			total++

			_, byID := have[ref{svc.Id, m.Name}]
			_, byName := have[ref{svc.Name, m.Name}]

			if byID || byName {
				covered++
			}
		}
	}

	return covered, total
}

func (h *RestServer) dashboardPayload(r *http.Request) rest.Dashboard {
	all := h.budgerigar.All()
	used := h.budgerigar.Used()
	services := h.collectAllServices()
	coveredMethods, totalMethods := methodCoverage(services, all)

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
		TotalServices:      len(services),
		TotalStubs:         len(all),
		UsedStubs:          len(used),
		UnusedStubs:        max(len(all)-len(used), 0),
		CoveredMethods:     coveredMethods,
		TotalMethods:       totalMethods,
		GrpcAddr:           h.ports.GRPC,
		GatewayAddr:        h.ports.Gateway,
		HttpAddr:           h.ports.HTTP,
		TotalSessions:      len(h.mergedSessions()),
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
