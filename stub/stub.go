package stub

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/bavix/gripmock/internal/app"
	"github.com/bavix/gripmock/internal/domain/rest"
)

type Options struct {
	Port     string
	BindAddr string
	StubPath string
}

const DefaultPort = "4771"

func RunRestServer(ch chan struct{}, opt Options) {
	if opt.Port == "" {
		opt.Port = DefaultPort
	}
	addr := net.JoinHostPort(opt.BindAddr, opt.Port)

	apiServer, _ := app.NewRestServer(opt.StubPath)

	router := mux.NewRouter()
	rest.HandlerFromMuxWithBaseURL(apiServer, router, "/api")

	fmt.Println("Serving stub admin on http://" + addr)
	go func() {
		handler := handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, router))
		// nosemgrep:go.lang.security.audit.net.use-tls.use-tls
		err := http.ListenAndServe(addr, handler)
		log.Fatal(err)
	}()

	go func() {
		select {
		case <-ch:
			apiServer.ServiceReady()
		}

		log.Println("gRPC-service is ready to accept requests")
	}()
}
