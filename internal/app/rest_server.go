package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/gripmock/stuber"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/pkg/grpcreflector"
	"github.com/bavix/gripmock/pkg/jsondecoder"
	"github.com/bavix/gripmock/pkg/yaml2json"
)

var (
	ErrServiceIsMissing = errors.New("service name is missing")
	ErrMethodIsMissing  = errors.New("method name is missing")
)

type RestServer struct {
	stuber    *stuber.Budgerigar
	convertor *yaml2json.Convertor
	caser     cases.Caser
	ok        atomic.Bool
	reflector *grpcreflector.GReflector
}

var _ rest.ServerInterface = &RestServer{}

func NewRestServer(path string, reflector *grpcreflector.GReflector) (*RestServer, error) {
	server := &RestServer{
		stuber:    stuber.NewBudgerigar(features.New(stuber.MethodTitle)),
		convertor: yaml2json.New(),
		caser:     cases.Title(language.English, cases.NoLower),
		reflector: reflector,
	}

	if path != "" {
		server.readStubs(path) // TODO: someday you will need to rewrite this code
	}

	return server, nil
}

func (h *RestServer) ServiceReady() {
	h.ok.Store(true)
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

	_ = json.NewEncoder(w).Encode(results)
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

	_ = json.NewEncoder(w).Encode(results)
}

func (h *RestServer) Liveness(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: time.Now()})
}

func (h *RestServer) Readiness(w http.ResponseWriter, _ *http.Request) {
	if !h.ok.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: time.Now()})
}

func (h *RestServer) AddStub(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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

	if err := json.NewEncoder(w).Encode(h.stuber.PutMany(inputs...)); err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	w.Header().Set("Content-Type", "application/json")
	h.stuber.DeleteByID(uuid)
}

func (h *RestServer) BatchStubsDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var inputs []uuid.UUID
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	defer r.Body.Close()

	if err := decoder.Decode(&inputs); err != nil {
		h.responseError(err, w)

		return
	}

	if len(inputs) > 0 {
		h.stuber.DeleteByID(inputs...)
	}
}

func (h *RestServer) ListUsedStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stuber.Used())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) ListUnusedStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stuber.Unused())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) ListStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stuber.All())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.stuber.Clear()

	w.WriteHeader(http.StatusNoContent)
}

func (h *RestServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query, err := stuber.NewQuery(r)
	if err != nil {
		h.responseError(err, w)

		return
	}

	defer r.Body.Close()

	result, err := h.stuber.FindByQuery(query)
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

	_ = json.NewEncoder(w).Encode(result.Found().Output)
}

func (h *RestServer) FindByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	stub := h.stuber.FindByID(uuid)
	if stub == nil {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	_ = json.NewEncoder(w).Encode(stub)
}

func (h *RestServer) responseError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(err, w)
}

func (h *RestServer) writeResponseError(err error, w http.ResponseWriter) {

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

func (h *RestServer) readStubs(path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Can't read stub from %s. %v\n", path, err)

		return
	}

	for _, file := range files {
		if file.IsDir() {
			h.readStubs(path + "/" + file.Name())

			continue
		}

		byt, err := os.ReadFile(path + "/" + file.Name())
		if err != nil {
			log.Printf("Error when reading file %s. %v. skipping...", file.Name(), err)

			continue
		}

		if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
			byt, err = h.convertor.Execute(file.Name(), byt)
			if err != nil {
				log.Printf("Error when unmarshalling file %s. %v. skipping...", file.Name(), err)

				continue
			}
		}

		var storageStubs []*stuber.Stub

		if err = jsondecoder.UnmarshalSlice(byt, &storageStubs); err != nil {
			log.Printf("Error when unmarshalling file %s. %v %v. skipping...", file.Name(), string(byt), err)

			continue
		}

		h.stuber.PutMany(storageStubs...)
	}
}

func validateStub(stub *stuber.Stub) error {
	if stub.Service == "" {
		return ErrServiceIsMissing
	}

	if stub.Method == "" {
		return ErrMethodIsMissing
	}

	switch {
	case stub.Input.Contains != nil:
		break
	case stub.Input.Equals != nil:
		break
	case stub.Input.Matches != nil:
		break
	default:
		// fixme
		//nolint:goerr113
		return fmt.Errorf("input cannot be empty")
	}

	// TODO: validate all input case

	if stub.Output.Error == "" && stub.Output.Data == nil && stub.Output.Code == nil {
		// fixme
		//nolint:goerr113
		return fmt.Errorf("output can't be empty")
	}

	return nil
}
