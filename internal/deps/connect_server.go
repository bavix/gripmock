package deps

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

const (
	connectReadHeaderTimeout = 10 * time.Second
	connectReadTimeout       = 30 * time.Second
	connectIdleTimeout       = 120 * time.Second
	connectMaxHeaderBytes    = 1 << 20
)

func (b *Builder) ConnectServe(ctx context.Context) error {
	gateway := b.newConnectGateway()

	router := mux.NewRouter()
	router.Handle("/{service}/{method}", gateway).Methods(http.MethodPost)

	srv := b.newConnectServer(ctx, router)

	listener, err := b.listenConnect(ctx, srv)
	if err != nil {
		return err
	}

	b.ender.Add(srv.Shutdown)

	zerolog.Ctx(ctx).Info().
		Str("addr", listener.Addr().String()).
		Bool("tls", srv.TLSConfig != nil).
		Msg("Serving ConnectRPC")

	return b.serveConnect(ctx, srv, listener)
}

func (b *Builder) newConnectGateway() *app.ConnectRPCGateway {
	var recorder history.Recorder
	if store := b.HistoryStore(); store != nil {
		recorder = store
	}

	return app.NewConnectRPCGateway(
		b.Budgerigar(),
		b.DescriptorRegistry(),
		recorder,
		nil,
		b.StubValidator(),
		b.ErrorFormatter(),
	)
}

func (b *Builder) newConnectServer(ctx context.Context, router *mux.Router) *http.Server {
	var handler http.Handler = router

	handler = httputil.GzipRequestMiddleware(handler)
	handler = handlers.CompressHandler(handler)

	if b.config.OtelEnabled {
		handler = otelhttp.NewHandler(handler, "gripmock-connect")
	}

	return &http.Server{
		Addr:              b.config.ConnectAddr,
		Handler:           handler,
		ReadHeaderTimeout: connectReadHeaderTimeout,
		ReadTimeout:       connectReadTimeout,
		IdleTimeout:       connectIdleTimeout,
		MaxHeaderBytes:    connectMaxHeaderBytes,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}

func (b *Builder) listenConnect(ctx context.Context, srv *http.Server) (net.Listener, error) {
	connectTLS := infraTLS.TLSConfig{
		CertFile:   b.config.ConnectTLSCertFile,
		KeyFile:    b.config.ConnectTLSKeyFile,
		ClientAuth: b.config.ConnectTLSClientAuth,
		CAFile:     b.config.ConnectTLSCAFile,
		MinVersion: infraTLS.MinTLSVersion12,
	}

	if connectTLS.IsEnabled() {
		return b.tlsConnectListener(srv, connectTLS)
	}

	srv.Protocols = func() *http.Protocols {
		var p http.Protocols
		p.SetHTTP1(true)
		p.SetUnencryptedHTTP2(true)

		return &p
	}()

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", b.config.ConnectAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to listen for ConnectRPC")
	}

	return listener, nil
}

func (b *Builder) tlsConnectListener(srv *http.Server, connectTLS infraTLS.TLSConfig) (net.Listener, error) {
	tlsCfg, err := connectTLS.BuildTLSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build Connect TLS config")
	}

	srv.TLSConfig = tlsCfg
	srv.Protocols = func() *http.Protocols {
		var p http.Protocols
		p.SetHTTP1(true)
		p.SetHTTP2(true)

		return &p
	}()

	tlsListener, err := tls.Listen("tcp", b.config.ConnectAddr, srv.TLSConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Connect TLS listener")
	}

	return tlsListener, nil
}

func (b *Builder) serveConnect(ctx context.Context, srv *http.Server, listener net.Listener) error {
	ch := make(chan error, 1)

	go func() {
		defer close(ch)

		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- err
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-ch:
		return errors.Wrap(err, "ConnectRPC server failed")
	}
}
