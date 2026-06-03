package deps

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

func (b *Builder) ConnectServe(ctx context.Context) error {
	var recorder history.Recorder
	if store := b.HistoryStore(); store != nil {
		recorder = store
	}

	gateway := app.NewConnectRPCGateway(
		b.Budgerigar(),
		b.DescriptorRegistry(),
		recorder,
		nil,
		b.StubValidator(),
		b.ErrorFormatter(),
	)

	mux := mux.NewRouter()
	mux.Handle("/{service}/{method}", gateway).Methods(http.MethodPost)

	const (
		readHeaderTimeout = 10 * time.Second
		readTimeout       = 30 * time.Second
		idleTimeout       = 120 * time.Second
		maxHeaderBytes    = 1 << 20
	)

	srv := &http.Server{
		Addr:              b.config.ConnectAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	connectTLS := infraTLS.TLSConfig{
		CertFile:   b.config.ConnectTLSCertFile,
		KeyFile:    b.config.ConnectTLSKeyFile,
		ClientAuth: b.config.ConnectTLSClientAuth,
		CAFile:     b.config.ConnectTLSCAFile,
		MinVersion: infraTLS.MinTLSVersion12,
	}

	var (
		listener net.Listener
		err      error
	)

	if connectTLS.IsEnabled() {
		tlsCfg, tlsErr := connectTLS.BuildTLSConfig()
		if tlsErr != nil {
			return errors.Wrap(tlsErr, "failed to build Connect TLS config")
		}

		srv.TLSConfig = tlsCfg

		tlsListener, tlsErr := tls.Listen("tcp", b.config.ConnectAddr, srv.TLSConfig)
		if tlsErr != nil {
			return errors.Wrap(tlsErr, "failed to create Connect TLS listener")
		}

		listener = tlsListener

		srv.Protocols = func() *http.Protocols {
			var p http.Protocols
			p.SetHTTP1(true)
			p.SetHTTP2(true)

			return &p
		}()
	} else {
		listener, err = (&net.ListenConfig{}).Listen(ctx, "tcp", b.config.ConnectAddr)
		if err != nil {
			return errors.Wrap(err, "failed to listen for ConnectRPC")
		}

		srv.Protocols = func() *http.Protocols {
			var p http.Protocols
			p.SetHTTP1(true)
			p.SetUnencryptedHTTP2(true)

			return &p
		}()
	}

	b.ender.Add(srv.Shutdown)

	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("addr", listener.Addr().String()).
		Bool("tls", connectTLS.IsEnabled()).
		Msg("Serving ConnectRPC")

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
