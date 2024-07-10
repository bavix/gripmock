//nolint:gochecknoglobals
package cmd

import (
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/internal/deps"
	"github.com/bavix/gripmock/internal/domain/waiter"
)

var (
	pingTimeout           time.Duration
	errServerIsNotRunning = errors.New("server is not running")
)

const serviceName = "gripmock"

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "The command checks whether the gripmock server is alive or dead by accessing it via the API",
	RunE: func(cmd *cobra.Command, _ []string) error {
		builder := deps.NewBuilder(deps.WithDefaultConfig())
		ctx, cancel := builder.SignalNotify(cmd.Context())
		defer cancel()

		ctx = builder.Logger(ctx)

		pingService, err := builder.PingService()
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("create ping service failed")

			return err
		}

		code, err := pingService.PingWithTimeout(ctx, pingTimeout, serviceName)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("unable to connect to server")

			return err
		}

		if code != waiter.Serving {
			zerolog.Ctx(ctx).Error().Uint32("code", uint32(code)).Msg("server is not running")

			return errServerIsNotRunning
		}

		return nil
	},
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(checkCmd)

	const defaultPingTimeout = time.Second * 5

	checkCmd.Flags().DurationVarP(&pingTimeout, "timeout", "t", defaultPingTimeout, "timeout")
}
