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
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

func (b *Builder) RestServe(
	ctx context.Context,
	stubPath string,
) (*http.Server, error) {
	extender := b.Extender()
	go extender.ReadFromPath(ctx, stubPath)

	apiServer, err := app.NewRestServer(ctx, b.Budgerigar(), extender)
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
			muxmiddleware.ContentType,
			muxmiddleware.RequestLogger,
		},
	})
	router.PathPrefix("/").Handler(http.FileServerFS(ui)).Methods(http.MethodGet)

	const timeout = time.Millisecond * 25

	srv := &http.Server{
		Addr:              b.config.HTTPAddr,
		ReadHeaderTimeout: timeout,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Handler: handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedHeaders([]string{
				"Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin",
				"X-GripMock-RequestInternal",
			}),
			handlers.AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodDelete}),
		)(router),
	}

	b.ender.Add(srv.Shutdown)

	return srv, nil
}
