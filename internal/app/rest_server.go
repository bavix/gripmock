package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/pkg/clock"
	"github.com/bavix/gripmock/pkg/storage"
	"github.com/bavix/gripmock/pkg/yaml2json"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type StubsServer struct {
	stubs     *storage.StubStorage
	convertor *yaml2json.Convertor
	caser     cases.Caser
	clock     *clock.Clock
}

func NewRestServer(path string) *StubsServer {
	server := &StubsServer{
		stubs:     storage.New(),
		convertor: yaml2json.New(),
		clock:     clock.New(),
		caser:     cases.Title(language.English, cases.NoLower),
	}

	if path != "" {
		server.readStubs(path) // TODO: someday you will need to rewrite this code
	}

	return server
}

// deprecated code
type findStubPayload struct {
	Service string                 `json:"service"`
	Method  string                 `json:"method"`
	Data    map[string]interface{} `json:"data"`
}

func (h *StubsServer) Liveness(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: h.clock.Now()})
}

func (h *StubsServer) Readiness(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(rest.MessageOK{Message: "ok", Time: h.clock.Now()})
}

func (h *StubsServer) AddStub(w http.ResponseWriter, r *http.Request) {
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

func (h *StubsServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, _ rest.ID) {
	w.Header().Set("Content-Type", "application/json")
	panic("DeleteStubByID")
}

func (h *StubsServer) ListUnusedStubs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	panic("ListUnusedStubs")
}

func (h *StubsServer) ListStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stubs.Stubs())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *StubsServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.stubs.Purge()
	w.WriteHeader(http.StatusNoContent)
}

func (h *StubsServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stub := new(findStubPayload)
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

	_ = json.NewEncoder(w).Encode(output)
}

func (h *StubsServer) responseError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(err, w)
}

func (h *StubsServer) writeResponseError(err error, w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

func (h *StubsServer) readStubs(path string) {
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
		//fixme
		//nolint:goerr113
		return fmt.Errorf("service name can't be empty")
	}

	if stub.Method == "" {
		return fmt.Errorf("method name can't be emtpy")
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
