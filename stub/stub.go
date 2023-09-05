package stub

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bavix/gripmock/internal/app"
	health_api "github.com/bavix/gripmock/pkg/api/health"
	stubs_api "github.com/bavix/gripmock/pkg/api/stubs"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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

	healthSrv, err := health_api.NewServer(&app.HealthcheckServer{})
	if err != nil {
		panic(err) // fixme: ...
	}

	stubsSrv, err := stubs_api.NewServer(app.NewStubsServer())
	if err != nil {
		panic(err) // fixme: ...
	}

	router := mux.NewRouter()
	router.Handle("/health", healthSrv)
	router.PathPrefix("/api").Subrouter().Handle("/", stubsSrv)

	fmt.Println("Serving stub admin on http://" + addr)
	go func() {
		handler := handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, router))
		err := http.ListenAndServe(addr, handler)
		log.Fatal(err)
	}()
}
