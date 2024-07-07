package stub

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gripmock/environment"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	gripmockui "github.com/bavix/gripmock-ui"
	"github.com/bavix/gripmock/internal/app"
	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/internal/pkg/muxmiddleware"
	"github.com/bavix/gripmock/pkg/grpcreflector"
)

func RunRestServer(
	ctx context.Context,
	stubPath string,
	config environment.Config,
	reflector *grpcreflector.GReflector,
) {
	const timeout = time.Millisecond * 25

	apiServer, _ := app.NewRestServer(stubPath, reflector)

	ui, _ := gripmockui.Assets()

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("gripmock-manager"))
	rest.HandlerWithOptions(apiServer, rest.GorillaServerOptions{
		BaseURL:     "/api",
		BaseRouter:  router,
		Middlewares: []rest.MiddlewareFunc{muxmiddleware.RequestLogger},
	})
	router.PathPrefix("/").Handler(http.FileServerFS(ui)).Methods(http.MethodGet)

	srv := &http.Server{
		Addr:              config.HTTPAddr,
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

	zerolog.Ctx(ctx).
		Info().
		Str("addr", config.HTTPAddr).
		Msg("stub-manager started")

	go func() {
		// nosemgrep:go.lang.security.audit.net.use-tls.use-tls
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zerolog.Ctx(ctx).Fatal().Err(err).Msg("stub manager completed")
		}
	}()

	<-ctx.Done()
}
