package cmd

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/internal/deps"
)

// restCmd represents the rest command
var restCmd = &cobra.Command{
	Use:   "rest",
	Short: "Start only the rest service",
	RunE: func(cmd *cobra.Command, args []string) error {
		builder := deps.NewBuilder(deps.WithDefaultConfig())
		ctx, cancel := builder.SignalNotify(cmd.Context())
		defer cancel()

		ctx = builder.Logger(ctx)

		return restServe(ctx, builder)
	},
}

func restServe(ctx context.Context, builder *deps.Builder) error {
	srv, err := builder.RestServe(ctx, stubFlag)
	if err != nil {
		return err
	}

	ch := make(chan error)
	defer close(ch)

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

func init() {
	rootCmd.AddCommand(restCmd)
}
