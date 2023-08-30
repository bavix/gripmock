package stub

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/tokopedia/gripmock/pkg/storage"
	"github.com/tokopedia/gripmock/pkg/yaml2json"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Handler struct {
	stubs     *storage.StubStorage
	convertor *yaml2json.Convertor
}

type findStubPayload struct {
	Service string                 `json:"service"`
	Method  string                 `json:"method"`
	Data    map[string]interface{} `json:"data"`
}

func NewHandler() *Handler {
	return &Handler{stubs: storage.New(), convertor: yaml2json.New()}
}

func (h *Handler) handleFind(w http.ResponseWriter, r *http.Request) {
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
		h.responseError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func (h *Handler) handlePurge(w http.ResponseWriter, _ *http.Request) {
	h.stubs.Purge()
	w.WriteHeader(204)
}

func (h *Handler) handleStubs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stubs.Stubs())
	if err != nil {
		h.responseError(err, w)
		return
	}
}

func (h *Handler) handleAddStub(w http.ResponseWriter, r *http.Request) {
	// todo: add supported input array
	stub := new(storage.Stub)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	if err := decoder.Decode(stub); err != nil {
		h.responseError(err, w)
		return
	}

	defer r.Body.Close()

	if err := validateStub(stub); err != nil {
		h.responseError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(h.stubs.Add(stub))
	if err != nil {
		h.responseError(err, w)
		return
	}
}

func (h *Handler) responseError(err error, w http.ResponseWriter) {
	w.WriteHeader(500)

	_, _ = w.Write([]byte(err.Error()))
}

func (h *Handler) readStubs(path string) {
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
