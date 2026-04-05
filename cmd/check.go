package cmd

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/v3/internal/deps"
)

var (
	pingTimeout  time.Duration //nolint:gochecknoglobals
	pingInterval time.Duration //nolint:gochecknoglobals
)

const serviceName = "gripmock"

var checkCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:          "check",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	Short:        "The command checks whether the gripmock server is alive or dead by accessing it via the API",
	RunE: func(cmd *cobra.Command, _ []string) error {
		builder := deps.NewBuilder(deps.WithDefaultConfig())

		ctx, cancel := builder.SignalNotify(cmd.Context())
		defer cancel()

		defer builder.Shutdown(context.WithoutCancel(ctx))

		ctx = builder.Logger(ctx)

		svc, err := builder.PingService()
		if err != nil {
			return errors.WithStack(err)
		}

		return svc.WaitForReady(ctx, pingTimeout, pingInterval, serviceName)
	},
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(checkCmd)

	const defaultPingTimeout = time.Second * 10

	const defaultPingInterval = time.Millisecond * 500

	checkCmd.Flags().DurationVarP(&pingTimeout, "timeout", "t", defaultPingTimeout, "total timeout to wait for server readiness")
	checkCmd.Flags().DurationVar(&pingInterval, "interval", defaultPingInterval, "interval between ping attempts")
	checkCmd.Flags().BoolVar(&checkCmd.SilenceErrors, "silent", false, "silence errors")
}
