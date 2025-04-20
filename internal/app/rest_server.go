package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/gripmock/stuber"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/pkg/jsondecoder"
)

var (
	ErrServiceIsMissing = errors.New("service name is missing")
	ErrMethodIsMissing  = errors.New("method name is missing")
)

type Extender interface {
	Wait(ctx context.Context)
}

type RestServer struct {
	ok         atomic.Bool
	budgerigar *stuber.Budgerigar
}

var _ rest.ServerInterface = &RestServer{}

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

	if err := json.NewEncoder(w).Encode(results); err != nil {
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

	if err := json.NewEncoder(w).Encode(results); err != nil {
		h.responseError(w, err)
	}
}

func (h *RestServer) liveness(w http.ResponseWriter) {
	if err := json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: time.Now()}); err != nil {
		h.responseError(w, err)
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
		h.responseError(w, err)

		return
	}

	defer r.Body.Close()

	var inputs []*stuber.Stub

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(w, err)

		return
	}

	for _, stub := range inputs {
		if err := validateStub(stub); err != nil {
			h.responseError(w, err)

			return
		}
	}

	if err := json.NewEncoder(w).Encode(h.budgerigar.PutMany(inputs...)); err != nil {
		h.responseError(w, err)

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
		h.responseError(w, err)

		return
	}

	defer r.Body.Close()

	var inputs []uuid.UUID

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(w, err)

		return
	}

	if len(inputs) > 0 {
		h.budgerigar.DeleteByID(inputs...)
	}
}

func (h *RestServer) ListUsedStubs(w http.ResponseWriter, _ *http.Request) {
	if err := json.NewEncoder(w).Encode(h.budgerigar.Used()); err != nil {
		h.responseError(w, err)
	}
}

func (h *RestServer) ListUnusedStubs(w http.ResponseWriter, _ *http.Request) {
	if err := json.NewEncoder(w).Encode(h.budgerigar.Unused()); err != nil {
		h.responseError(w, err)
	}
}

func (h *RestServer) ListStubs(w http.ResponseWriter, _ *http.Request) {
	if err := json.NewEncoder(w).Encode(h.budgerigar.All()); err != nil {
		h.responseError(w, err)
	}
}

func (h *RestServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.budgerigar.Clear()

	w.WriteHeader(http.StatusNoContent)
}

func (h *RestServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	query, err := stuber.NewQuery(r)
	if err != nil {
		h.responseError(w, err)

		return
	}

	defer r.Body.Close()

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

func (h *RestServer) responseError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(w, err)
}

func (h *RestServer) writeResponseError(w http.ResponseWriter, err error) {
	h.writeResponse(w, map[string]string{
		"error": err.Error(),
	})
}

func (h *RestServer) writeResponse(w http.ResponseWriter, data any) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.responseError(w, err)
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
		return errors.New("input cannot be empty")
	}

	if stub.Output.Error == "" && stub.Output.Data == nil && stub.Output.Code == nil {
		return errors.New("output cannot be empty")
	}

	return nil
}
