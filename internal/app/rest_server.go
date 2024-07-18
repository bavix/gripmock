package app

import (
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gripmock/json"
	"github.com/gripmock/stuber"

	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/pkg/grpcreflector"
	"github.com/bavix/gripmock/pkg/jsondecoder"
)

var (
	ErrServiceIsMissing = errors.New("service name is missing")
	ErrMethodIsMissing  = errors.New("method name is missing")
)

type Extender interface {
	Wait()
}

type RestServer struct {
	ok         atomic.Bool
	budgerigar *stuber.Budgerigar
	reflector  *grpcreflector.GReflector
}

var _ rest.ServerInterface = &RestServer{}

func NewRestServer(
	budgerigar *stuber.Budgerigar,
	extender Extender,
	reflector *grpcreflector.GReflector,
) (*RestServer, error) {
	server := &RestServer{
		reflector:  reflector,
		budgerigar: budgerigar,
	}

	go func() {
		if extender != nil {
			extender.Wait()
		}

		server.ok.Store(true)
	}()

	return server, nil
}

func (h *RestServer) ServicesList(w http.ResponseWriter, r *http.Request) {
	services, err := h.reflector.Services(r.Context())
	if err != nil {
		return
	}

	results := make([]rest.Service, len(services))

	for i, service := range services {
		serviceMethods, err := h.reflector.Methods(r.Context(), service.ID)
		if err != nil {
			continue
		}

		restMethods := make([]rest.Method, len(serviceMethods))
		for i, serviceMethod := range serviceMethods {
			restMethods[i] = rest.Method{
				Id:   serviceMethod.ID,
				Name: serviceMethod.Name,
			}
		}

		results[i] = rest.Service{
			Id:      service.ID,
			Name:    service.Name,
			Package: service.Package,
			Methods: restMethods,
		}
	}

	if err := json.Encode(w, results); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) ServiceMethodsList(w http.ResponseWriter, r *http.Request, serviceID string) {
	methods, err := h.reflector.Methods(r.Context(), serviceID)
	if err != nil {
		return
	}

	results := make([]rest.Method, len(methods))
	for i, method := range methods {
		results[i] = rest.Method{
			Id:   method.ID,
			Name: method.Name,
		}
	}

	if err := json.Encode(w, results); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) liveness(w http.ResponseWriter) {
	if err := json.Encode(w, rest.MessageOK{Message: "ok", Time: time.Now()}); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) Readiness(w http.ResponseWriter, _ *http.Request) {
	if !h.ok.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	h.liveness(w)
}

func (h *RestServer) Liveness(w http.ResponseWriter, _ *http.Request) {
	h.liveness(w)
}

func (h *RestServer) AddStub(w http.ResponseWriter, r *http.Request) {
	byt, err := io.ReadAll(r.Body)
	if err != nil {
		h.responseError(err, w)

		return
	}

	defer r.Body.Close()

	var inputs []*stuber.Stub

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(err, w)

		return
	}

	for _, stub := range inputs {
		if err := validateStub(stub); err != nil {
			h.responseError(err, w)

			return
		}
	}

	if err := json.Encode(w, h.budgerigar.PutMany(inputs...)); err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	h.budgerigar.DeleteByID(uuid)

	w.WriteHeader(http.StatusNoContent)
}

func (h *RestServer) BatchStubsDelete(w http.ResponseWriter, r *http.Request) {
	byt, err := io.ReadAll(r.Body)
	if err != nil {
		h.responseError(err, w)

		return
	}

	defer r.Body.Close()

	var inputs []uuid.UUID

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(err, w)

		return
	}

	if len(inputs) > 0 {
		h.budgerigar.DeleteByID(inputs...)
	}
}

func (h *RestServer) ListUsedStubs(w http.ResponseWriter, _ *http.Request) {
	if err := json.Encode(w, h.budgerigar.Used()); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) ListUnusedStubs(w http.ResponseWriter, _ *http.Request) {
	if err := json.Encode(w, h.budgerigar.Unused()); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) ListStubs(w http.ResponseWriter, _ *http.Request) {
	if err := json.Encode(w, h.budgerigar.All()); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.budgerigar.Clear()

	w.WriteHeader(http.StatusNoContent)
}

func (h *RestServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	query, err := stuber.NewQuery(r)
	if err != nil {
		h.responseError(err, w)

		return
	}

	defer r.Body.Close()

	result, err := h.budgerigar.FindByQuery(query)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(err, w)

		return
	}

	if result.Found() == nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(stubNotFoundError2(query, result), w)

		return
	}

	if err := json.Encode(w, result.Found().Output); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) FindByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	stub := h.budgerigar.FindByID(uuid)
	if stub == nil {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err := json.Encode(w, stub); err != nil {
		h.responseError(err, w)
	}
}

func (h *RestServer) responseError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(err, w)
}

func (h *RestServer) writeResponseError(err error, w http.ResponseWriter) {
	if err := json.Encode(w, map[string]string{
		"error": err.Error(),
	}); err != nil {
		h.responseError(err, w)
	}
}

// validateStub validates if the stub is valid or not.
func validateStub(stub *stuber.Stub) error {
	if stub.Service == "" {
		return ErrServiceIsMissing
	}

	if stub.Method == "" {
		return ErrMethodIsMissing
	}

	if stub.Input.Contains == nil && stub.Input.Equals == nil && stub.Input.Matches == nil {
		return errors.New("input cannot be empty") //nolint:err113
	}

	if stub.Output.Error == "" && stub.Output.Data == nil && stub.Output.Code == nil {
		return errors.New("output cannot be empty") //nolint:err113
	}

	return nil
}
