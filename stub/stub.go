package stub

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/tokopedia/gripmock/pkg/storage"
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

	api := NewHandler()
	if opt.StubPath != "" {
		api.readStubs(opt.StubPath)
	}

	router := mux.NewRouter()

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/stubs/search", api.searchHandle).Methods("POST")
	apiRouter.HandleFunc("/stubs", api.listHandle).Methods("GET")
	apiRouter.HandleFunc("/stubs", api.addHandle).Methods("POST")
	apiRouter.HandleFunc("/stubs", api.purgeHandle).Methods("DELETE")

	fmt.Println("Serving stub admin on http://" + addr)
	go func() {
		http.Handle("/", router)
		err := http.ListenAndServe(addr, nil)
		log.Fatal(err)
	}()
}

func validateStub(stub *storage.Stub) error {
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
