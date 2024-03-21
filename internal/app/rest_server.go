package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/internal/pkg/features"
	"github.com/bavix/gripmock/internal/pkg/grpcreflector"
	"github.com/bavix/gripmock/pkg/clock"
	"github.com/bavix/gripmock/pkg/storage"
	"github.com/bavix/gripmock/pkg/yaml2json"
)

var (
	ErrServiceIsMissing = errors.New("service name is missing")
	ErrMethodIsMissing  = errors.New("method name is missing")
)

type RestServer struct {
	stubs     *storage.StubStorage
	convertor *yaml2json.Convertor
	caser     cases.Caser
	clock     *clock.Clock
	ok        atomic.Bool
	reflector *grpcreflector.GReflector
}

var _ rest.ServerInterface = &RestServer{}

func NewRestServer(path string, reflector *grpcreflector.GReflector) (*RestServer, error) {
	stubsStorage, err := storage.New()
	if err != nil {
		return nil, err
	}

	server := &RestServer{
		stubs:     stubsStorage,
		convertor: yaml2json.New(),
		clock:     clock.New(),
		caser:     cases.Title(language.English, cases.NoLower),
		reflector: reflector,
	}

	if path != "" {
		server.readStubs(path) // TODO: someday you will need to rewrite this code
	}

	return server, nil
}

// deprecated code.
type findStubPayload struct {
	ID       *uuid.UUID             `json:"id,omitempty"`
	Service  string                 `json:"service"`
	Method   string                 `json:"method"`
	Headers  map[string]interface{} `json:"headers"`
	Data     map[string]interface{} `json:"data"`
	features features.FeatureSlice
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
		results[i] = rest.Service{
			Id:      service.ID,
			Name:    service.Name,
			Package: service.Package,
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
			Id:        method.ID,
			ServiceId: method.Service.ID,
			Package:   method.Service.Package,
			Name:      method.Name,
		}
	}

	_ = json.NewEncoder(w).Encode(results)
}

func (h *RestServer) Liveness(w http.ResponseWriter, _ *http.Request) {
	//nolint:errchkjson
	_ = json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: h.clock.Now()})
}

func (h *RestServer) Readiness(w http.ResponseWriter, _ *http.Request) {
	if !h.ok.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	//nolint:errchkjson
	_ = json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: h.clock.Now()})
}

func (h *RestServer) AddStub(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	byt, err := io.ReadAll(r.Body)
	if err != nil {
		h.responseError(err, w)

		return
	}

	defer r.Body.Close()

	byt = bytes.TrimSpace(byt)

	if byt[0] == '{' && byt[len(byt)-1] == '}' {
		byt = []byte("[" + string(byt) + "]")
	}

	var inputs []*storage.Stub
	decoder := json.NewDecoder(bytes.NewReader(byt))
	decoder.UseNumber()

	if err := decoder.Decode(&inputs); err != nil {
		h.responseError(err, w)

		return
	}

	for _, stub := range inputs {
		if err := validateStub(stub); err != nil {
			h.responseError(err, w)

			return
		}
	}

	if err := json.NewEncoder(w).Encode(h.stubs.Add(inputs...)); err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	w.Header().Set("Content-Type", "application/json")
	h.stubs.Delete(uuid)
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
		h.stubs.Delete(inputs...)
	}
}

func (h *RestServer) ListUsedStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stubs.Used())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) ListUnusedStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stubs.Unused())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) ListStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stubs.Stubs())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *RestServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.stubs.Purge()
	w.WriteHeader(http.StatusNoContent)
}

func (h *RestServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stub := &findStubPayload{features: features.New(r)}
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	if err := decoder.Decode(stub); err != nil {
		h.responseError(err, w)

		return
	}

	defer r.Body.Close()

	// due to golang implementation
	// method name must capital
	title := cases.Title(language.English, cases.NoLower)
	stub.Method = title.String(stub.Method)

	output, err := findStub(h.stubs, stub)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Println(err)
		h.writeResponseError(err, w)

		return
	}

	//nolint:errchkjson
	_ = json.NewEncoder(w).Encode(output)
}

func (h *RestServer) FindByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	stub := h.stubs.FindByID(uuid)
	if stub == nil {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	//nolint:errchkjson
	_ = json.NewEncoder(w).Encode(stub)
}

func (h *RestServer) responseError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(err, w)
}

func (h *RestServer) writeResponseError(err error, w http.ResponseWriter) {
	//nolint:errchkjson
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

//nolint:cyclop
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

		byt = bytes.TrimSpace(byt)

		if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
			byt, err = h.convertor.Execute(file.Name(), byt)
			if err != nil {
				log.Printf("Error when unmarshalling file %s. %v. skipping...", file.Name(), err)

				continue
			}
		}

		if byt[0] == '{' && byt[len(byt)-1] == '}' {
			byt = []byte("[" + string(byt) + "]")
		}

		var storageStubs []*storage.Stub
		decoder := json.NewDecoder(bytes.NewReader(byt))
		decoder.UseNumber()

		if err = decoder.Decode(&storageStubs); err != nil {
			log.Printf("Error when unmarshalling file %s. %v %v. skipping...", file.Name(), string(byt), err)

			continue
		}

		h.stubs.Add(storageStubs...)
	}
}

func validateStub(stub *storage.Stub) error {
	if stub.Service == "" {
		return ErrServiceIsMissing
	}

	if stub.Method == "" {
		return ErrMethodIsMissing
	}

	// due to golang implementation
	// method name must capital
	title := cases.Title(language.English, cases.NoLower)
	stub.Method = title.String(stub.Method)

	switch {
	case stub.Input.Contains != nil:
		break
	case stub.Input.Equals != nil:
		break
	case stub.Input.Matches != nil:
		break
	default:
		//fixme
		//nolint:goerr113
		return fmt.Errorf("input cannot be empty")
	}

	// TODO: validate all input case

	if stub.Output.Error == "" && stub.Output.Data == nil && stub.Output.Code == nil {
		//fixme
		//nolint:goerr113
		return fmt.Errorf("output can't be empty")
	}

	return nil
}
