package deps

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/bavix/gripmock/v3/internal/app"
	modern "github.com/bavix/gripmock/v3/internal/infra/http/modern"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
	"github.com/bavix/gripmock/v3/internal/infra/repository"
	"github.com/bavix/gripmock/v3/internal/infra/store/memory"
)

//nolint:funlen
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
	// Legacy API - mount individual handlers to avoid interface conflicts
	legacyRouter := router.PathPrefix("/api").Subrouter()
	legacyRouter.Use(
		muxmiddleware.PanicRecoveryMiddleware,
		muxmiddleware.ContentType,
		muxmiddleware.RequestLogger,
	)

	// Mount legacy endpoints manually
	legacyRouter.HandleFunc("/stubs", apiServer.AddStub).Methods(http.MethodPost)
	legacyRouter.HandleFunc("/stubs", apiServer.ListStubs).Methods(http.MethodGet)
	legacyRouter.HandleFunc("/stubs/search", apiServer.SearchStubs).Methods(http.MethodPost)
	legacyRouter.HandleFunc("/stubs/batchDelete", apiServer.BatchStubsDelete).Methods(http.MethodPost)
	legacyRouter.HandleFunc("/stubs/used", apiServer.ListUsedStubs).Methods(http.MethodGet)
	legacyRouter.HandleFunc("/stubs/unused", apiServer.ListUnusedStubs).Methods(http.MethodGet)
	legacyRouter.HandleFunc("/stubs/purge", apiServer.PurgeStubs).Methods(http.MethodPost)
	legacyRouter.HandleFunc("/services", apiServer.ServicesList).Methods(http.MethodGet)
	// Note: ServiceMethodsList requires path parameters, handled separately if needed
	legacyRouter.HandleFunc("/ready", apiServer.Readiness).Methods(http.MethodGet)
	legacyRouter.HandleFunc("/live", apiServer.Liveness).Methods(http.MethodGet)

	analyticsRepo := b.Analytics()

	// Create history repository with config from environment
	historyRepo := memory.NewInMemoryHistory(
		b.config.HistoryLimit.Int64(),
		strings.Join(b.config.HistoryRedactKeys, ","),
	)

	// Create a stub repository that uses the same Budgerigar as legacy API
	stubRepo := repository.NewStubRepository(b.Budgerigar())
	v4Server := modern.NewServer(stubRepo, analyticsRepo, historyRepo, b.PluginInfos())
	v4Server.Mount(router, "/api/v4")

	// Add metrics endpoint with Go runtime metrics
	router.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

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
