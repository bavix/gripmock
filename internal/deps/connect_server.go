package deps

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
)

// ConnectServe starts the ConnectRPC HTTP server and blocks until ctx is done.
func (b *Builder) ConnectServe(ctx context.Context) error {
	var recorder history.Recorder
	if store := b.HistoryStore(); store != nil {
		recorder = store
	}

	handler := app.NewConnectHandler(
		b.Budgerigar(),
		b.DescriptorRegistry(),
		recorder,
	)

	const (
		readHeaderTimeout = 10 * time.Second
		readTimeout       = 30 * time.Second
		writeTimeout      = 30 * time.Second
		idleTimeout       = 120 * time.Second
		maxHeaderBytes    = 1 << 20
	)

	srv := &http.Server{
		Addr:              b.config.ConnectAddr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	b.ender.Add(srv.Shutdown)

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", b.config.ConnectAddr)
	if err != nil {
		return errors.Wrap(err, "failed to listen for ConnectRPC")
	}

	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("addr", listener.Addr().String()).
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
