package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/v3/internal/domain/rest"
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
}

// RestServer implements legacy REST API endpoints only
// V4 API is handled by v4.Server

// NewRestServer creates a new REST server instance with the specified dependencies.
func NewRestServer(
	ctx context.Context,
	budgerigar *stuber.Budgerigar,
	extender Extender,
) (*RestServer, error) {
	server := &RestServer{
		budgerigar: budgerigar,
	}

	go func() {
		if extender != nil {
			extender.Wait(ctx)
		}

		server.ok.Store(true)
	}()

	return server, nil
}

// BatchDeleteStubsV4 is a placeholder method to satisfy the interface
// This method is not used in legacy API.
func (h *RestServer) BatchDeleteStubsV4(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Method not supported in legacy API", http.StatusMethodNotAllowed)
}

// ClearHistoryV4 is a placeholder method to satisfy the interface
// This method is not used in legacy API.
func (h *RestServer) ClearHistoryV4(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Method not supported in legacy API", http.StatusMethodNotAllowed)
}

// ServicesList returns a list of all available gRPC services.
func (h *RestServer) ServicesList(w http.ResponseWriter, _ *http.Request) {
	results := make([]rest.Service, 0)

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

	err := json.NewEncoder(w).Encode(results)
	if err != nil {
		h.responseError(w, err)
	}
}

func splitLast(s string, sep string) []string {
	lastDot := strings.LastIndex(s, sep)
	if lastDot == -1 {
		return []string{s, ""}
	}

	return []string{s[:lastDot], s[lastDot+1:]}
}

// ServiceMethodsList returns a list of methods for the specified service.
func (h *RestServer) ServiceMethodsList(w http.ResponseWriter, _ *http.Request, serviceID string) {
	results := make([]rest.Method, 0)

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

	err := json.NewEncoder(w).Encode(results)
	if err != nil {
		h.responseError(w, err)
	}
}

// Readiness handles the readiness probe endpoint.
func (h *RestServer) Readiness(w http.ResponseWriter, _ *http.Request) {
	if !h.ok.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	h.liveness(w)
}

// Liveness handles the liveness probe endpoint.
func (h *RestServer) Liveness(w http.ResponseWriter, _ *http.Request) {
	h.liveness(w)
}

// AddStub adds a new stub to the server.
func (h *RestServer) AddStub(w http.ResponseWriter, r *http.Request) {
	byt, err := io.ReadAll(r.Body)
	if err != nil {
		h.responseError(w, err)

		return
	}

	defer func() {
		_ = r.Body.Close()
	}()

	var inputs []*stuber.Stub

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(w, err)

		return
	}

	for _, stub := range inputs {
		if err := validateStub(stub); err != nil {
			h.validationError(w, err)

			return
		}
	}

	if err := json.NewEncoder(w).Encode(h.budgerigar.PutMany(inputs...)); err != nil {
		h.responseError(w, err)

		return
	}
}

// DeleteStubByID deletes a stub by its ID.
func (h *RestServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	h.budgerigar.DeleteByID(uuid)

	w.WriteHeader(http.StatusNoContent)
}

// BatchStubsDelete deletes multiple stubs in a batch operation.
func (h *RestServer) BatchStubsDelete(w http.ResponseWriter, r *http.Request) {
	byt, err := io.ReadAll(r.Body)
	if err != nil {
		h.responseError(w, err)

		return
	}

	defer func() {
		_ = r.Body.Close()
	}()

	var inputs []uuid.UUID

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(w, err)

		return
	}

	if len(inputs) > 0 {
		h.budgerigar.DeleteByID(inputs...)
	}
}

// ListUsedStubs returns a list of stubs that have been used.
func (h *RestServer) ListUsedStubs(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(h.budgerigar.Used()); err != nil {
		h.responseError(w, err)
	}
}

// ListUnusedStubs returns a list of stubs that have not been used.
func (h *RestServer) ListUnusedStubs(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(h.budgerigar.Unused()); err != nil {
		h.responseError(w, err)
	}
}

// ListStubs returns a list of all stubs.
func (h *RestServer) ListStubs(w http.ResponseWriter, _ *http.Request) {
	if err := json.NewEncoder(w).Encode(h.budgerigar.All()); err != nil {
		h.responseError(w, err)
	}
}

// PurgeStubs clears all stubs from the server.
func (h *RestServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.budgerigar.Clear()

	w.WriteHeader(http.StatusNoContent)
}

// SearchStubs searches for stubs based on a query.
func (h *RestServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	query, err := stuber.NewQuery(r)
	if err != nil {
		h.responseError(w, err)

		return
	}

	defer func() {
		_ = r.Body.Close()
	}()

	result, err := h.budgerigar.FindByQuery(query)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(w, err)

		return
	}

	if result.Found() == nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(w, stubNotFoundError(query, result))

		return
	}

	if err := json.NewEncoder(w).Encode(result.Found().Output); err != nil {
		h.responseError(w, err)
	}
}

// FindByID retrieves a stub by its ID.
func (h *RestServer) FindByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	stub := h.budgerigar.FindByID(uuid)
	if stub == nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponse(w, map[string]string{
			"error": fmt.Sprintf("Stub with ID '%s' not found", uuid),
		})

		return
	}

	if err := json.NewEncoder(w).Encode(stub); err != nil {
		h.responseError(w, err)
	}
}

// liveness handles the liveness probe response.
func (h *RestServer) liveness(w http.ResponseWriter) {
	if err := json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: time.Now()}); err != nil {
		h.responseError(w, err)
	}
}

// responseError writes an error response to the HTTP writer.
func (h *RestServer) responseError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(w, err)
}

// validationError writes a validation error response to the HTTP writer.
func (h *RestServer) validationError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)

	h.writeResponseError(w, err)
}

// writeResponseError writes an error response to the HTTP writer.
func (h *RestServer) writeResponseError(w http.ResponseWriter, err error) {
	h.writeResponse(w, map[string]string{
		"error": err.Error(),
	})
}

// writeResponse writes a successful response to the HTTP writer.
func (h *RestServer) writeResponse(w http.ResponseWriter, data any) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.responseError(w, err)
	}
}

// validateStub validates if the stub is valid or not.
func validateStub(stub *stuber.Stub) error {
	validate := validator.New()

	// Register custom validation functions
	if err := validate.RegisterValidation("valid_input_config", validateInputConfiguration); err != nil {
		return err
	}

	if err := validate.RegisterValidation("valid_output_config", validateOutputConfiguration); err != nil {
		return err
	}

	// Create a validation struct with tags
	vStub := &validationStub{
		Service: stub.Service,
		Method:  stub.Method,
		Input:   stub.Input,
		Inputs:  stub.Inputs,
		Output:  stub.Output,
	}

	if err := validate.Struct(vStub); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, fieldError := range validationErrors {
				return &ValidationError{
					Field:   fieldError.Field(),
					Tag:     fieldError.Tag(),
					Value:   fieldError.Value(),
					Message: getValidationMessage(fieldError),
				}
			}
		}

		return err
	}

	// Additional validation of stub structure
	if err := validateStubStructure(stub); err != nil {
		return err
	}

	return nil
}
