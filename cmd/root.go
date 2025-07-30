package cmd

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/v3/internal/deps"
	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

var (
	stubFlag           string
	importsFlag        []string
	streamIntervalFlag string
	version            = "development"
)

var rootCmd = &cobra.Command{
	Use:     "gripmock",
	Short:   "gRPC Mock Server",
	Version: version,
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamInterval, err := time.ParseDuration(streamIntervalFlag)
		if err != nil {
			return errors.Wrapf(err, "invalid stream-interval: %s", streamIntervalFlag)
		}

		builder := deps.NewBuilder(deps.WithDefaultConfig())
		ctx, cancel := builder.SignalNotify(cmd.Context())
		defer cancel()

		ctx = builder.Logger(ctx)

		zerolog.Ctx(ctx).Info().
			Str("release", version).
			Dur("stream_interval", streamInterval).
			Msg("Starting GripMock")

		go func() {
			if err := restServe(ctx, builder); err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("Failed to start rest server")
			}
		}()

		defer builder.Shutdown(context.WithoutCancel(ctx))

		return builder.GRPCServe(ctx, proto.New(args, importsFlag), streamInterval)
	},
}

func restServe(ctx context.Context, builder *deps.Builder) error {
	srv, err := builder.RestServe(ctx, stubFlag)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Str("addr", srv.Addr).Msg("HTTP server is now running")

	ch := make(chan error)

	go func() {
		defer close(ch)

		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				ch <- ctx.Err()
			}

			return
		case ch <- srv.ListenAndServe():
			return
		}
	}()

	if err := <-ch; !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "http server failed")
	}

	return nil
}

func init() {
	rootCmd.Flags().StringVarP(
		&stubFlag,
		"stub",
		"s",
		"",
		"Path where the stub files are (Optional)")

	rootCmd.Flags().StringSliceVarP(
		&importsFlag,
		"imports",
		"i",
		[]string{},
		"Path to import proto-libraries")

	rootCmd.Flags().StringVar(
		&streamIntervalFlag,
		"stream-interval",
		"100ms",
		"Interval between stream messages (e.g., 100ms, 1s, 500ms)")
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
