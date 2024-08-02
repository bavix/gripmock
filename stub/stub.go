package stub

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/internal/app"
	"github.com/bavix/gripmock/internal/domain/rest"
	"github.com/bavix/gripmock/internal/pkg/muxmiddleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

type Options struct {
	Port       string
	BindAddr   string
	StubPath   string
	StrictMode bool
}

func RunRestServer(ctx context.Context, ch chan struct{}, opt Options) {
	const timeout = time.Millisecond * 25

	addr := net.JoinHostPort(opt.BindAddr, opt.Port)

	apiServer, _ := app.NewRestServer(opt.StubPath, opt.StrictMode)

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
		Handler: router,
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
