package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/internal/config"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Version: version,
	Use:     "gripmock",
	Short:   "gRPC Mock Server\n\n",
	Long: `GripMock is a mock server for gRPC services. 
It's using a .proto file to generate implementation of gRPC service for you. 
You can use gripmock for setting up end-to-end testing or as a dummy server in a software development phase. 
The server implementation is in GoLang but the client can be any programming language that support gRPC.`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println(args,
			cmd.Flag("output").Value,
			cmd.Flag("stub").Value,
			cmd.Flag("imports").Value,
			cmd.Flag("grpc-port").Value,
			cmd.Flag("grpc-listen").Value,
			cmd.Flag("admin-port").Value,
			cmd.Flag("admin-listen").Value,
		)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context, conf config.Config) {
	rootCmd.Flags().String("output", "", "directory to output server.go. Default is $GOPATH/src/grpc/")

	rootCmd.Flags().String("stub", "", "Path where the stub files are (Optional)")
	rootCmd.Flags().String("imports", "/protobuf,/googleapis", "comma separated imports path. default path /protobuf,/googleapis is where gripmock Dockerfile install WKT protos") //nolint:lll

	// deprecated. will be removed in 3.x
	rootCmd.Flags().String("grpc-port", conf.GRPC.Port, "Deprecated: use ENV GRPC_PORT. Port of gRPC tcp server")                                                                                    //nolint:lll
	rootCmd.Flags().String("grpc-listen", conf.GRPC.Host, "Deprecated: use ENV GRPC_HOST. Address the gRPC server will bind to. Default to localhost, set to 0.0.0.0 to use from another machine")   //nolint:lll
	rootCmd.Flags().String("admin-port", conf.HTTP.Port, "Deprecated: use ENV HTTP_PORT. Port of stub admin server")                                                                                 //nolint:lll
	rootCmd.Flags().String("admin-listen", conf.HTTP.Host, "Deprecated: use ENV HTTP_HOST. Address the admin server will bind to. Default to localhost, set to 0.0.0.0 to use from another machine") //nolint:lll

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
