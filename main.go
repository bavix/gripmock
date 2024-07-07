package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/gripmock/grpc-interceptors"
	"github.com/rs/zerolog"
	_ "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	_ "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	_ "github.com/bavix/gripmock-sdk-go"
	"github.com/bavix/gripmock/internal/pkg/patcher"
	"github.com/bavix/gripmock/pkg/dependencies"
	"github.com/bavix/gripmock/stub"
)

var version = "development"

//nolint:funlen,cyclop
func main() {
	outputPointer := flag.String("output", "", "directory to output server.go. Default is $GOPATH/src/grpc/")
	flag.StringVar(outputPointer, "o", *outputPointer, "alias for -output")

	stubPath := flag.String("stub", "", "Path where the stub files are (Optional)")                                                                                                //nolint:lll,staticcheck
	imports := flag.String("imports", "/protobuf,/googleapis", "comma separated imports path. default path /protobuf,/googleapis is where gripmock Dockerfile install WKT protos") //nolint:lll,staticcheck

	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	builder, err := dependencies.New(ctx, "gripmock-rest")
	if err != nil {
		log.Fatal(err) //nolint:gocritic
	}

	logger := builder.Logger()

	ctx = logger.WithContext(ctx)

	// parse proto files
	protoPaths := flag.Args()

	// Ensure at least one proto file is provided
	if len(protoPaths) == 0 {
		logger.Fatal().Msg("at least one proto file is required")
	}

	// Start GripMock server
	//nolint:godox
	// TODO: move validation of required arguments to a separate service
	logger.Info().Str("release", version).Msg("Starting GripMock")

	// Check if $GOPATH is set
	if os.Getenv("GOPATH") == "" {
		logger.Fatal().Msg("$GOPATH is empty")
	}

	// Set output directory
	output := *outputPointer
	if output == "" {
		// Default to $GOPATH/src/grpc if output is not provided
		output = os.Getenv("GOPATH") + "/src/grpc"
	}

	// For safety
	output += "/"

	// Check if output folder exists, if not create it
	// nosemgrep:semgrep-go.os-error-is-not-exist
	if _, err := os.Stat(output); os.IsNotExist(err) {
		// Create output folder
		if err := os.Mkdir(output, os.ModePerm); err != nil {
			logger.Fatal().Err(err).Msg("unable to create output folder")
		}
	}

	chReady := make(chan struct{})
	defer close(chReady)

	// Run the admin stub server in a separate goroutine.
	//
	// This goroutine runs the REST server that serves the stub files.
	// It waits for the ready signal from the gRPC server goroutine.
	// Once the gRPC server is ready, it starts the admin stub server.
	go func() {
		<-chReady

		zerolog.Ctx(ctx).Info().Msg("gRPC server is ready to accept requests")

		stub.RunRestServer(ctx, *stubPath, builder.Config(), builder.Reflector())
	}()

	importDirs := strings.Split(*imports, ",")

	// Generate protoc-generated code and run the gRPC server.
	//
	// This section generates the protoc-generated code (pb.go) and runs the gRPC server.
	// It creates the output directory if it does not exist.
	// It then generates the protoc-generated code using the protocParam struct.
	// Finally, it runs the gRPC server using the runGrpcServer function.
	generateProtoc(ctx, protocParam{
		protoPath: protoPaths,
		output:    output,
		imports:   importDirs,
	})

	// And run
	run, chErr := runGrpcServer(ctx, output)

	// Wait for the gRPC server to start and confirm that it is in the "SERVING" state.
	// This is done by checking the health check service on the server.
	// If the service is in the "SERVING" state, it means that the server has started successfully.
	go func() {
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		waiter := healthv1.NewHealthClient(builder.GRPCClient())

		// Check the health of the server.
		// The empty string in the request means that we want to check the whole server,
		// not a specific service.
		// The grpc.WaitForReady(true) parameter means that we want to wait for the server to become ready.
		check, err := waiter.Check(ctx, &healthv1.HealthCheckRequest{Service: ""}, grpc.WaitForReady(true))
		if err != nil {
			return
		}

		// If the server is in the "SERVING" state, send a signal to the chReady channel.
		if check.GetStatus() == healthv1.HealthCheckResponse_SERVING {
			chReady <- struct{}{}
		}
	}()

	// Wait for the gRPC server to exit or the context to be done.
	select {
	case err := <-chErr:
		// If the gRPC server exits with an error, log the error.
		logger.Fatal().Err(err).Msg("gRPC server exited with an error")
	case <-ctx.Done():
		// If the context is done, check if there was an error.
		if err := ctx.Err(); err != nil {
			logger.Err(err).Msg("an error has occurred")
		}

		// Log that the gRPC server is stopping.
		logger.Info().Msg("Stopping gRPC Server")

		// Kill the gRPC server process.
		if err := run.Process.Kill(); err != nil {
			logger.Fatal().Err(err).Msg("failed to kill process")
		}
	}
}

// protocParam represents the parameters for the protoc command.
type protocParam struct {
	// protoPath is a list of paths to the proto files.
	protoPath []string

	// output is the output directory for the generated files.
	output string

	// imports is a list of import paths.
	imports []string
}

// getProtodirs returns a list of proto directories based on the given protoPath
// and imports.
//
// It takes a context.Context and a protoPath string as well as a slice of strings
// representing the imports. The protoPath string is used to deduce the proto
// directory, and the imports are used to search for a proto directory prefix.
//
// The function returns a slice of strings representing the proto directories.
func getProtodirs(_ context.Context, protoPath string, imports []string) []string {
	// Deduce the proto directory from the proto path.
	splitPath := strings.Split(protoPath, "/")
	protoDir := ""

	// If there are any elements in splitPath, join them up to the second-to-last
	// element with path.Join to get the proto directory.
	if len(splitPath) > 0 {
		protoDir = path.Join(splitPath[:len(splitPath)-1]...)
	}

	// Search for the proto directory prefix in the imports.
	protoDirIdx := -1

	for i := range imports {
		// Join the "protogen" directory with the import directory to get the full
		// directory path.
		dir := path.Join("protogen", imports[i])

		// If the proto directory starts with the full directory path, set the proto
		// directory to the full directory path and set the index of the proto directory
		// in the imports slice.
		if strings.HasPrefix(protoDir, dir) {
			protoDir = dir
			protoDirIdx = i

			break
		}
	}

	// Create a slice to hold the proto directories.
	protoDirs := make([]string, 0, len(imports)+1)

	// Append the proto directory to the slice.
	protoDirs = append(protoDirs, protoDir)

	// Loop through the imports and append each directory to the slice, skipping
	// any directories that have already been added.
	for i, dir := range imports {
		if i == protoDirIdx {
			continue
		}

		protoDirs = append(protoDirs, dir)
	}

	// Return the slice of proto directories.
	return protoDirs
}

// generateProtoc is a function that runs the protoc command with the given
// parameters.
//
// It takes a context.Context and a protocParam struct as parameters. The
// protocParam struct contains the protoPath, output, and imports fields that
// are used to configure the protoc command.
//
// It generates the protoc command arguments and runs the protoc command with
// the given parameters. If there is an error running the protoc command, it
// logs a fatal message.
func generateProtoc(ctx context.Context, param protocParam) {
	// Fix the go_package option for each proto file in the protoPath.
	param.protoPath = fixGoPackage(ctx, param.protoPath)

	// Get the proto directories based on the protoPath and imports.
	protodirs := getProtodirs(ctx, param.protoPath[0], param.imports)

	// Estimate the length of the args slice to prevent expanding it.
	args := make([]string, 0, len(protodirs)+len(param.protoPath)+2) //nolint:mnd

	// Append the -I option for each proto directory to the args slice.
	for _, dir := range protodirs {
		args = append(args, "-I", dir)
	}

	// Set the output directory for generated files to $GOPATH/src.
	pbOutput := os.Getenv("GOPATH") + "/src"

	// Append the protoPath, --go_out, --go-grpc_out, and --gripmock_out options
	// to the args slice.
	args = append(args, param.protoPath...)
	args = append(args, "--go_out="+pbOutput)
	args = append(args, "--go-grpc_out="+pbOutput)
	args = append(args, "--gripmock_out="+param.output)

	// Create a new exec.Cmd command with the protoc command and the args.
	protoc := exec.Command("protoc", args...)

	// Set the environment variables for the command.
	protoc.Env = os.Environ()

	// Set the stdout and stderr for the command.
	protoc.Stdout = os.Stdout
	protoc.Stderr = os.Stderr

	// Run the protoc command and log a fatal message if there is an error.
	if err := protoc.Run(); err != nil {
		zerolog.Ctx(ctx).Fatal().Err(err).Msg("fail on protoc")
	}
}

// fixGoPackage is a function that appends the go_package option to each
// proto file in the given protoPaths if the proto file doesn't already have
// one.
//
// It reads each proto file, creates a temporary file with the go_package option,
// and copies the contents of the original file to the temporary file. The
// temporary file is then returned as part of the results.
//
// ctx is the context.Context to use for the function.
// protoPaths is a slice of string paths to the proto files.
// fixGoPackage returns a slice of string paths to the temporary files.
func fixGoPackage(ctx context.Context, protoPaths []string) []string {
	results := make([]string, 0, len(protoPaths))

	for _, protoPath := range protoPaths {
		pile, err := os.OpenFile(protoPath, os.O_RDONLY, 0o600) //nolint:mnd
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msgf("unable to open protofile %s", protoPath)

			continue
		}

		defer pile.Close()

		packageName := "protogen/" + strings.Trim(filepath.Dir(protoPath), "/")

		if err := os.MkdirAll(packageName, 0o666); err != nil { //nolint:mnd
			zerolog.Ctx(ctx).Err(err).Msgf("unable to create temp dir %s", protoPath)

			continue
		}

		tmp, err := os.Create(filepath.Join(packageName, filepath.Base(protoPath)))
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msgf("unable to create temp file %s", protoPath)

			continue
		}

		defer tmp.Close()

		if _, err = io.Copy(patcher.NewWriterWrapper(tmp, packageName), pile); err != nil {
			zerolog.Ctx(ctx).Err(err).Msgf("unable to copy file %s", protoPath)

			continue
		}

		results = append(results, tmp.Name())
	}

	return results
}

// runGrpcServer runs the gRPC server in a separate process.
//
// ctx is the context.Context to use for the command.
// output is the output directory where the server.go file is located.
// It returns the exec.Cmd object representing the running process, and a channel
// that receives an error when the process exits.
func runGrpcServer(ctx context.Context, output string) (*exec.Cmd, <-chan error) {
	// Construct the command to run the gRPC server.
	run := exec.CommandContext(ctx, "go", "run", output+"server.go") //nolint:gosec
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
