package deps

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

//nolint:funlen
func (b *Builder) RestServe(
	ctx context.Context,
	stubPath string,
) (*http.Server, error) {
	extender := b.Extender(ctx)
	go extender.ReadFromPath(ctx, stubPath)

	var historyReader history.Reader
	if store := b.HistoryStore(); store != nil {
		historyReader = store
	}

	apiServer, err := app.NewRestServer(ctx, b.Budgerigar(), extender, historyReader, b.StubValidator())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create rest server")
	}

	ui, err := b.ui()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get UI assets")
	}

	router := mux.NewRouter()
	rest.HandlerWithOptions(apiServer, rest.GorillaServerOptions{
		BaseURL:    "/api",
		BaseRouter: router,
		Middlewares: []rest.MiddlewareFunc{
			httputil.MaxBodySize(httputil.MaxBodyBytes()),
			muxmiddleware.PanicRecoveryMiddleware,
			muxmiddleware.ContentType,
			muxmiddleware.RequestLogger,
		},
	})
	router.PathPrefix("/").Handler(http.FileServerFS(ui)).Methods(http.MethodGet)

	const (
		readHeaderTimeout = 10 * time.Second
		readTimeout       = 30 * time.Second
		writeTimeout      = 30 * time.Second
		idleTimeout       = 120 * time.Second
		maxHeaderBytes    = 1 << 20
	)

	srv := &http.Server{
		Addr:              b.config.HTTPAddr,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Handler: handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedHeaders([]string{
				"Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin",
				"X-GripMock-RequestInternal",
				"X-Gripmock-Session",
			}),
			handlers.AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch}),
		)(router),
	}

	b.ender.Add(srv.Shutdown)

	return srv, nil
}
