package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"io/fs"
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

	if len(protoPaths) == 0 {
		logger.Fatal().Msg("at least one proto file is required")
	}

	//nolint:godox
	// fixme: move validation of required arguments to a separate service
	logger.Info().Str("release", version).Msg("Starting GripMock")

	if os.Getenv("GOPATH") == "" {
		logger.Fatal().Msg("$GOPATH is empty")
	}

	output := *outputPointer
	if output == "" {
		output = os.Getenv("GOPATH") + "/src/grpc"
	}

	// for safety
	output += "/"
	if _, err := os.Stat(output); errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(output, os.ModePerm); err != nil {
			logger.Fatal().Err(err).Msg("can't create output folder")
		}
	}

	chReady := make(chan struct{})
	defer close(chReady)

	// run admin stub server
	stub.RunRestServer(ctx, chReady, *stubPath, builder.Config(), builder.Reflector())

	importDirs := strings.Split(*imports, ",")

	// generate pb.go and grpc server based on proto
	generateProtoc(ctx, protocParam{
		protoPath: protoPaths,
		output:    output,
		imports:   importDirs,
	})

	// and run
	run, chErr := runGrpcServer(ctx, output)

	// This is a kind of crutch, but now there is no other solution.
	// I have an idea to combine gripmock and grpcmock services into one, then this check will be easier to do.
	// Checking the grpc port of the service. If the port appears, the service has started successfully.
	go func() {
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		waiter := healthv1.NewHealthClient(builder.GRPCClient())

		check, err := waiter.Check(ctx, &healthv1.HealthCheckRequest{Service: ""}, grpc.WaitForReady(true))
		if err != nil {
			return
		}

		if check.GetStatus() == healthv1.HealthCheckResponse_SERVING {
			chReady <- struct{}{}
		}
	}()

	select {
	case err := <-chErr:
		log.Fatal(err)
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			logger.Err(err).Msg("an error has occurred")
		}

		logger.Info().Msg("Stopping gRPC Server")

		if err := run.Process.Kill(); err != nil {
			logger.Fatal().Err(err).Msg("process killed")
		}
	}
}

type protocParam struct {
	protoPath []string
	output    string
	imports   []string
}

func getProtodirs(_ context.Context, protoPath string, imports []string) []string {
	// deduced proto dir from proto path
	splitPath := strings.Split(protoPath, "/")
	protoDir := ""

	if len(splitPath) > 0 {
		protoDir = path.Join(splitPath[:len(splitPath)-1]...)
	}

	// search protoDir prefix
	protoDirIdx := -1

	for i := range imports {
		dir := path.Join("protogen", imports[i])
		if strings.HasPrefix(protoDir, dir) {
			protoDir = dir
			protoDirIdx = i

			break
		}
	}

	protoDirs := make([]string, 0, len(imports)+1)
	protoDirs = append(protoDirs, protoDir)
	// include all dir in imports, skip if it has been added before
	for i, dir := range imports {
		if i == protoDirIdx {
			continue
		}

		protoDirs = append(protoDirs, dir)
	}

	return protoDirs
}

func generateProtoc(ctx context.Context, param protocParam) {
	param.protoPath = fixGoPackage(ctx, param.protoPath)
	protodirs := getProtodirs(ctx, param.protoPath[0], param.imports)

	// estimate args length to prevent expand
	args := make([]string, 0, len(protodirs)+len(param.protoPath)+2) //nolint:mnd
	for _, dir := range protodirs {
		args = append(args, "-I", dir)
	}

	// the latest go-grpc plugin will generate subfolders under $GOPATH/src based on go_package option
	pbOutput := os.Getenv("GOPATH") + "/src"

	args = append(args, param.protoPath...)
	args = append(args, "--go_out="+pbOutput)
	args = append(args, "--go-grpc_out="+pbOutput)
	args = append(args, "--gripmock_out="+param.output)
	protoc := exec.Command("protoc", args...)
	protoc.Env = os.Environ()
	protoc.Stdout = os.Stdout
	protoc.Stderr = os.Stderr

	if err := protoc.Run(); err != nil {
		zerolog.Ctx(ctx).Fatal().Err(err).Msg("fail on protoc")
	}
}

// append gopackage in proto files if doesn't have any.
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

func runGrpcServer(ctx context.Context, output string) (*exec.Cmd, <-chan error) {
	run := exec.CommandContext(ctx, "go", "run", output+"server.go") //nolint:gosec
	run.Env = os.Environ()
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr

	err := run.Start()
	if err != nil {
		zerolog.Ctx(ctx).Fatal().Err(err).Msg("unable to start grpc service")
	}

	zerolog.Ctx(ctx).Info().Int("pid", run.Process.Pid).Msg("gRPC-service started")

	runErr := make(chan error)

	go func() {
		runErr <- run.Wait()
	}()

	return run, runErr
}
