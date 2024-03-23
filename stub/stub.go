package stub

import (
	"context"
	"errors"
	"github.com/bavix/gripmock/internal/pkg/grpcreflector"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"github.com/bavix/gripmock/internal/app"
	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/internal/pkg/features"
	"github.com/bavix/gripmock/internal/pkg/muxmiddleware"
)

type Options struct {
	Port     string
	BindAddr string
	StubPath string
}

func RunRestServer(ctx context.Context, ch chan struct{}, opt Options, reflector *grpcreflector.GReflector) {
	const timeout = time.Millisecond * 25

	addr := net.JoinHostPort(opt.BindAddr, opt.Port)

	apiServer, _ := app.NewRestServer(opt.StubPath, reflector)

	router := mux.NewRouter()
	router.Use(muxmiddleware.RequestLogger)
	router.Use(otelmux.Middleware("gripmock-manager"))
	rest.HandlerFromMuxWithBaseURL(apiServer, router, "/api")

	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: timeout,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Handler: handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedHeaders([]string{
				"Accept", "Accept-Language", "Content-Type", "Content-Language", "Origin",
				string(features.RequestInternal),
			}),
			handlers.AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodDelete}),
		)(router),
	}

	zerolog.Ctx(ctx).
		Info().
		Str("addr", addr).
		Msg("stub-manager started")

	go func() {
		// nosemgrep:go.lang.security.audit.net.use-tls.use-tls
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zerolog.Ctx(ctx).Fatal().Err(err).Msg("stub manager completed")
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			apiServer.ServiceReady()
		}

		zerolog.Ctx(ctx).Info().Msg("gRPC-service is ready to accept requests")
	}()
}
