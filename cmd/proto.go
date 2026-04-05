package cmd

import (
	"github.com/spf13/cobra"
)

func init() { //nolint:gochecknoinits
	protoCmd := &cobra.Command{
		Use:   "proto",
		Short: "Proto file utilities",
	}

	rootCmd.AddCommand(protoCmd)

	registerProtoExport(protoCmd)
}
