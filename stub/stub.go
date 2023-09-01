package stub

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/bavix/gripmock/pkg/storage"
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

	api := NewApiHandler()
	if opt.StubPath != "" {
		api.readStubs(opt.StubPath)
	}

	healthcheck := NewHealthcheckHandler()

	router := mux.NewRouter()
	router.Handle("/health", healthcheck).Methods("GET")

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/stubs/search", api.SearchHandle).Methods("POST")
	apiRouter.HandleFunc("/stubs", api.ListHandle).Methods("GET")
	apiRouter.HandleFunc("/stubs", api.AddHandle).Methods("POST")
	apiRouter.HandleFunc("/stubs", api.PurgeHandle).Methods("DELETE")

	fmt.Println("Serving stub admin on http://" + addr)
	go func() {
		handler := handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, router))
		err := http.ListenAndServe(addr, handler)
		log.Fatal(err)
	}()
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

	if stub.Output.Error == "" && stub.Output.Data == nil {
		//fixme
		//nolint:goerr113
		return fmt.Errorf("output can't be empty")
	}

	return nil
}
