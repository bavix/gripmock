// Package httpmock provides a httptest server using the real gripmock REST handlers.
// Shares Budgerigar and MemoryStore for testing remote mode scenarios.
package httpmock

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Server wraps httptest.Server with real gripmock REST handlers.
type Server struct {
	HTTPServer *httptest.Server
	URL        string
	Budgerigar *stuber.Budgerigar
	Recorder   *history.MemoryStore
	RestServer *app.RestServer
}

// NewServer creates a test server backed by the real gripmock REST API.
func NewServer() *Server {
	b := stuber.NewBudgerigar()
	r := history.NewMemoryStore(0)

	restSrv, err := app.NewRestServer(
		context.Background(),
		b,
		app.NewInstantExtender(),
		r,
		nil, // validator — auto-created
		nil, // descriptor registry — auto-created
		nil, // error formatter — auto-created
	)
	if err != nil {
		panic("httpmock: failed to create RestServer: " + err.Error())
	}

	router := mux.NewRouter()
	rest.HandlerFromMuxWithBaseURL(restSrv, router, "/api")

	// Apply gzip decompression middleware (supports gzip, deflate, zstd, snappy, br)
	var handler http.Handler = router

	handler = httputil.GzipRequestMiddleware(handler)

	hs := httptest.NewServer(handler)

	return &Server{
		HTTPServer: hs,
		URL:        hs.URL,
		Budgerigar: b,
		Recorder:   r,
		RestServer: restSrv,
	}
}

// Close shuts down the test server.
func (s *Server) Close() {
	s.HTTPServer.Close()
}

// RecordCall adds a call to the history recorder.
func (s *Server) RecordCall(service, method string, request, response map[string]any) {
	s.Recorder.Record(history.CallRecord{
		Service:  service,
		Method:   method,
		Request:  request,
		Response: response,
	})
}
