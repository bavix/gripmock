package stub

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"net/http"
)

type Options struct {
	Port     string
	BindAddr string
	StubPath string
}

const DefaultPort = "4771"

func RunStubServer(opt Options) {
	if opt.Port == "" {
		opt.Port = DefaultPort
	}
	addr := opt.BindAddr + ":" + opt.Port
	r := chi.NewRouter()
	r.Post("/add", addStub)
	r.Get("/", listStub)
	r.Post("/find", handleFindStub)
	r.Get("/clear", handleClearStub)

	if opt.StubPath != "" {
		readStubFromFile(opt.StubPath)
	}

	fmt.Println("Serving stub admin on http://" + addr)
	go func() {
		err := http.ListenAndServe(addr, r)
		log.Fatal(err)
	}()
}

func responseError(err error, w http.ResponseWriter) {
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}

type Stub struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Input   Input  `json:"input"`
	Output  Output `json:"output"`
}

type Input struct {
	Equals   map[string]interface{} `json:"equals"`
	Contains map[string]interface{} `json:"contains"`
	Matches  map[string]interface{} `json:"matches"`
}

type Output struct {
	Data  map[string]interface{} `json:"data"`
	Error string                 `json:"error"`
}

func addStub(w http.ResponseWriter, r *http.Request) {
	stub := new(Stub)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	if err := decoder.Decode(stub); err != nil {
		responseError(err, w)
		return
	}

	defer r.Body.Close()

	if err := validateStub(stub); err != nil {
		responseError(err, w)
		return
	}

	if err := storeStub(stub); err != nil {
		responseError(err, w)
		return
	}

	_, err := w.Write([]byte("Success add stub"))
	if err != nil {
		responseError(err, w)
		return
	}
}

func listStub(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allStub())
}

func validateStub(stub *Stub) error {
	if stub.Service == "" {
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
		return fmt.Errorf("input cannot be empty")
	}

	// TODO: validate all input case

	if stub.Output.Error == "" && stub.Output.Data == nil {
		return fmt.Errorf("output can't be empty")
	}
	return nil
}

type findStubPayload struct {
	Service string                 `json:"service"`
	Method  string                 `json:"method"`
	Data    map[string]interface{} `json:"data"`
}

func handleFindStub(w http.ResponseWriter, r *http.Request) {
	stub := new(findStubPayload)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	if err := decoder.Decode(stub); err != nil {
		responseError(err, w)
		return
	}

	defer r.Body.Close()

	// due to golang implementation
	// method name must capital
	title := cases.Title(language.English, cases.NoLower)
	stub.Method = title.String(stub.Method)

	output, err := findStub(stub)
	if err != nil {
		log.Println(err)
		responseError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func handleClearStub(w http.ResponseWriter, r *http.Request) {
	clearStorage()
	w.Write([]byte("OK"))
}
