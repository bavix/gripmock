//nolint:gochecknoglobals
package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/internal/deps"
	"github.com/bavix/gripmock/internal/domain/proto"
)

var (
	outputFlag  string
	stubFlag    string
	importsFlag []string
	version     = "development"
)

var rootCmd = &cobra.Command{
	Use:     "gripmock",
	Short:   "gRPC Mock Server",
	Version: version,
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		builder := deps.NewBuilder(deps.WithDefaultConfig())
		ctx, cancel := builder.SignalNotify(cmd.Context())
		defer cancel()

		ctx = builder.Logger(ctx)

		zerolog.Ctx(ctx).Info().Str("release", version).Msg("Starting GripMock")

		go func() {
			if err := restServe(ctx, builder); err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("failed to start rest server")
			}
		}()

		return builder.GRPCServe(cmd.Context(), proto.NewProtocParam(args, outputFlag, importsFlag))
	},
}

func restServe(ctx context.Context, builder *deps.Builder) error {
	srv, err := builder.RestServe(ctx, stubFlag)
	if err != nil {
		return err
	}

	ch := make(chan error)

	go func() {
		zerolog.Ctx(ctx).
			Info().
			Str("addr", builder.Config().HTTPAddr).
			Msg("Serving stub-manager")

		ch <- srv.ListenAndServe()
	}()

	select {
	case err = <-ch:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

//nolint:gochecknoinits
func init() {
	rootCmd.Flags().StringVarP(
		&outputFlag,
		"output",
		"o",
		os.Getenv("GOPATH")+"/src/grpc",
		"Server generation directory server.go")

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
		[]string{"/protobuf", "/googleapis"},
		"Path to import proto-libraries")
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
