package stub

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/bavix/gripmock/pkg/storage"
	"github.com/bavix/gripmock/pkg/yaml2json"
)

type HealthcheckHandler struct{}

func (*HealthcheckHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func NewHealthcheckHandler() *HealthcheckHandler {
	return &HealthcheckHandler{}
}

type ApiHandler struct {
	stubs     *storage.StubStorage
	convertor *yaml2json.Convertor
}

type findStubPayload struct {
	Service string                 `json:"service"`
	Method  string                 `json:"method"`
	Data    map[string]interface{} `json:"data"`
}

func NewApiHandler() *ApiHandler {
	return &ApiHandler{stubs: storage.New(), convertor: yaml2json.New()}
}

func (h *ApiHandler) SearchHandle(w http.ResponseWriter, r *http.Request) {
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
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(err, w)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(output)
}

func (h *ApiHandler) PurgeHandle(w http.ResponseWriter, _ *http.Request) {
	h.stubs.Purge()
	w.WriteHeader(http.StatusNoContent)
}

func (h *ApiHandler) ListHandle(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stubs.Stubs())
	if err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *ApiHandler) AddHandle(w http.ResponseWriter, r *http.Request) {
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

	var stubs []*storage.Stub
	decoder := json.NewDecoder(bytes.NewReader(byt))
	decoder.UseNumber()

	if err := decoder.Decode(&stubs); err != nil {
		h.responseError(err, w)

		return
	}

	for _, stub := range stubs {
		if err := validateStub(stub); err != nil {
			h.responseError(err, w)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.stubs.Add(stubs...)); err != nil {
		h.responseError(err, w)

		return
	}
}

func (h *ApiHandler) responseError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)

	h.writeResponseError(err, w)
}

func (h *ApiHandler) writeResponseError(err error, w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

func (h *ApiHandler) readStubs(path string) {
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

		var stubs []*storage.Stub
		decoder := json.NewDecoder(bytes.NewReader(byt))
		decoder.UseNumber()

		if err = decoder.Decode(&stubs); err != nil {
			log.Printf("Error when unmarshalling file %s. %v %v. skipping...", file.Name(), string(byt), err)

			continue
		}

		h.stubs.Add(stubs...)
	}
}
