package cmd

import (
	"context"
	"net/http"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/v3/internal/deps"
	"github.com/bavix/gripmock/v3/internal/domain/proto"
	"github.com/bavix/gripmock/v3/internal/infra/build"
)

var (
	stubFlag    string   //nolint:gochecknoglobals
	importsFlag []string //nolint:gochecknoglobals
	pluginsFlag []string //nolint:gochecknoglobals
)

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:     "gripmock",
	Short:   "gRPC Mock Server",
	Version: build.Version,
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		builder := deps.NewBuilder(
			deps.WithDefaultConfig(),
			deps.WithPlugins(pluginsFlag),
		)

		ctx, cancel := builder.SignalNotify(cmd.Context())
		defer cancel()

		ctx = builder.Logger(ctx)
		builder.LoadPlugins(ctx)

		zerolog.Ctx(ctx).Info().
			Str("release", build.Version).
			Int("pid", os.Getpid()).
			Msg("Starting GripMock")

		go func() {
			defer func() {
				if r := recover(); r != nil {
					zerolog.Ctx(ctx).
						Fatal().
						Interface("panic", r).
						Msg("Fatal panic in REST server goroutine - terminating server")
				}
			}()

			if err := restServe(ctx, builder); err != nil {
				zerolog.Ctx(ctx).Fatal().Err(err).Msg("Fatal error in REST server - terminating server")
			}
		}()

		defer builder.Shutdown(context.WithoutCancel(ctx))

		return builder.GRPCServe(ctx, proto.New(args, importsFlag))
	},
}

func restServe(ctx context.Context, builder *deps.Builder) error {
	srv, err := builder.RestServe(ctx, stubFlag)
	if err != nil {
		return errors.Wrap(err, "failed to start rest server")
	}

	zerolog.Ctx(ctx).Info().Str("addr", srv.Addr).Msg("HTTP server is now running")

	ch := make(chan error)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				zerolog.Ctx(ctx).
					Fatal().
					Interface("panic", r).
					Msg("Fatal panic in HTTP server goroutine - terminating server")
			}
		}()
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

func init() { //nolint:gochecknoinits
	rootCmd.PersistentFlags().StringVarP(
		&stubFlag,
		"stub",
		"s",
		"",
		"Path where the stub files are (Optional)")

	rootCmd.PersistentFlags().StringSliceVarP(
		&importsFlag,
		"imports",
		"i",
		[]string{},
		"Path to import proto-libraries")

	rootCmd.PersistentFlags().StringSliceVar(
		&pluginsFlag,
		"plugins",
		[]string{},
		"Template plugin paths (.so)")
}

// Execute runs the root command with the given context.
func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
