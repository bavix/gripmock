package app

import (
	"context"
	"os"
	"os/exec"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/internal/domain/proto"
	"github.com/bavix/gripmock/internal/domain/servergen"
)

type GRPCServer struct {
	params *proto.ProtocParam
}

func NewGRPCServer(params *proto.ProtocParam) *GRPCServer {
	return &GRPCServer{params}
}

func (s *GRPCServer) Serve(ctx context.Context) error {
	err := servergen.ServerGenerate(ctx, s.params)
	if err != nil {
		return err
	}

	server, ch := s.newServer(ctx)

	// Wait for the gRPC server to exit or the context to be done.
	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		// If the context is done, check if there was an error.
		if err := ctx.Err(); err != nil {
			return err
		}

		// Kill the gRPC server process.
		if err := server.Process.Kill(); err != nil {
			return err
		}
	}

	return nil
}

// newServer runs the gRPC server in a separate process.
//
// ctx is the context.Context to use for the command.
// output is the output directory where the server.go file is located.
// It returns the exec.Cmd object representing the running process, and a channel
// that receives an error when the process exits.
func (s *GRPCServer) newServer(ctx context.Context) (*exec.Cmd, <-chan error) {
	// Construct the command to run the gRPC server.
	run := exec.CommandContext(ctx, "go", "run", s.params.Output()+"/server.go") //nolint:gosec
	run.Env = os.Environ()
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr

	// Start the command.
	if err := run.Start(); err != nil {
		zerolog.Ctx(ctx).Fatal().Err(err).Msg("unable to start gRPC service")
	}

	// Log the process ID.
	zerolog.Ctx(ctx).Info().Int("pid", run.Process.Pid).Msg("gRPC-service started")

	// Create a channel to receive the process exit error.
	runErr := make(chan error)

	// Start a goroutine to wait for the process to exit and send the error
	// to the channel.
	go func() {
		runErr <- run.Wait()
	}()

	return run, runErr
}
