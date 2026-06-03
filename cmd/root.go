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
	sourceFlag  []string //nolint:gochecknoglobals
)

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:     "gripmock",
	Short:   "gRPC Mock Server",
	Version: build.Version + " (" + build.Commit + ") " + build.Date,
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		builder := deps.NewBuilder(
			deps.WithDefaultConfig(),
			deps.WithPlugins(pluginsFlag),
		)

		ctx, cancel := builder.SignalNotify(cmd.Context())
		defer cancel()

		ctx = builder.Logger(ctx)
		builder.InitTelemetry(ctx)
		builder.LoadPlugins(ctx)

		zerolog.Ctx(ctx).Info().
			Str("release", build.Version).
			Str("commit", build.Commit).
			Str("date", build.Date).
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

		go func() {
			defer func() {
				if r := recover(); r != nil {
					zerolog.Ctx(ctx).
						Fatal().
						Interface("panic", r).
						Msg("Fatal panic in ConnectRPC server goroutine - terminating server")
				}
			}()

			if err := builder.ConnectServe(ctx); err != nil {
				zerolog.Ctx(ctx).Fatal().Err(err).Msg("Fatal error in ConnectRPC server - terminating server")
			}
		}()

		defer builder.Shutdown(context.WithoutCancel(ctx))

		// Parse arguments with per-proxy source bindings
		// This uses raw os.Args to detect -S flag positioning relative to proxy URLs
		params := proto.ParseArgumentsWithBindings(args, importsFlag, sourceFlag)

		zerolog.Ctx(ctx).Info().
			Strs("args", args).
			Strs("sourceFlag", sourceFlag).
			Msg("Starting GRPCServe")

		return builder.GRPCServe(ctx, params)
	},
}

func restServe(ctx context.Context, builder *deps.Builder) error {
	srv, err := builder.RestServe(ctx, stubFlag)
	if err != nil {
		return errors.Wrap(err, "failed to start rest server")
	}

	zerolog.Ctx(ctx).Info().Str("addr", srv.Addr()).Bool("tls", srv.TLSEnabled()).Msg("HTTP server is now running")

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

	rootCmd.PersistentFlags().StringSliceVarP(
		&sourceFlag,
		"source",
		"S",
		[]string{},
		"Local descriptor sources for proxy modes (.proto, .protoset, .pb, directory)")
}

// Execute runs the root command with the given context.
func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
