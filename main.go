package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	_ "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	_ "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	_ "github.com/bavix/gripmock-sdk-go"
	"github.com/bavix/gripmock/internal/config"
	"github.com/bavix/gripmock/internal/container"
	"github.com/bavix/gripmock/internal/pkg/patcher"
	"github.com/bavix/gripmock/pkg/trace"
	"github.com/bavix/gripmock/pkg/utils"
	"github.com/bavix/gripmock/stub"
)

var version string

//nolint:funlen,cyclop
func main() {
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logLevel, err := zerolog.ParseLevel(conf.App.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	logger := utils.NewLogger(logLevel)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	ctx = logger.WithContext(ctx)

	if err := trace.InitTracer(ctx, "gripmock", conf.OTLPTrace); err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("connect to tracer")
	}

	outputPointer := flag.String("output", "", "directory to output server.go. Default is $GOPATH/src/grpc/")
	flag.StringVar(outputPointer, "o", *outputPointer, "alias for -output")

	stubPath := flag.String("stub", "", "Path where the stub files are (Optional)")
	imports := flag.String("imports", "/protobuf,/googleapis", "comma separated imports path. default path /protobuf,/googleapis is where gripmock Dockerfile install WKT protos") //nolint:lll

	flag.Parse()

	box := container.New(&conf)

	reflector, err := box.GReflector(ctx)
	if err != nil {
		logger.Fatal().Msg("reflector is required")
	}

	// parse proto files
	protoPaths := flag.Args()

	if len(protoPaths) == 0 {
		logger.Fatal().Msg("at least one proto file is required")
	}

	// for backwards compatibility
	if os.Args[1] == "gripmock" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
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
	stub.RunRestServer(ctx, chReady, stub.Options{
		StubPath: *stubPath,
		Port:     conf.HTTP.Port,
		BindAddr: conf.HTTP.Host,
	}, reflector)

	importDirs := strings.Split(*imports, ",")

	// generate pb.go and grpc server based on proto
	generateProtoc(ctx, protocParam{
		protoPath:       protoPaths,
		adminHost:       conf.HTTP.Host,
		adminPort:       conf.HTTP.Port,
		grpcNet:         conf.GRPC.Network,
		grpcAddress:     conf.GRPC.Host,
		grpcPort:        conf.GRPC.Port,
		otlpHost:        conf.OTLPTrace.Host,
		otlpPort:        conf.OTLPTrace.Port,
		otlpTLS:         conf.OTLPTrace.TLS,
		otlpSampleRatio: conf.OTLPTrace.SampleRatio,
		output:          output,
		imports:         importDirs,
	})

	// and run
	run, chErr := runGrpcServer(ctx, output)

	// This is a kind of crutch, but now there is no other solution.
	// I have an idea to combine gripmock and grpcmock services into one, then this check will be easier to do.
	// Checking the grpc port of the service. If the port appears, the service has started successfully.
	go func() {
		var d net.Dialer

		for {
			dialCtx, cancel := context.WithTimeout(ctx, time.Second)

			conn, err := d.DialContext(dialCtx, conf.GRPC.Network, conf.GRPCAddr())

			cancel()

			if err == nil && conn != nil {
				chReady <- struct{}{}

				conn.Close()

				break
			}
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
	protoPath       []string
	adminHost       string
	adminPort       string
	grpcAddress     string
	grpcNet         string
	grpcPort        string
	output          string
	otlpHost        string
	otlpPort        string
	otlpTLS         bool
	otlpSampleRatio float64
	imports         []string
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
	args := make([]string, 0, len(protodirs)+len(param.protoPath)+2) //nolint:gomnd
	for _, dir := range protodirs {
		args = append(args, "-I", dir)
	}

	// the latest go-grpc plugin will generate subfolders under $GOPATH/src based on go_package option
	pbOutput := os.Getenv("GOPATH") + "/src"

	gmOut := []string{
		fmt.Sprintf(
			"admin-host=%s,admin-port=%s",
			param.adminHost, param.adminPort),
		fmt.Sprintf("grpc-network=%s,grpc-address=%s,grpc-port=%s",
			param.grpcNet, param.grpcAddress, param.grpcPort),
		fmt.Sprintf("otlp-host=%s,otlp-port=%s,otlp-tls=%d,otlp-ratio=%f",
			param.otlpHost, param.otlpPort, bool2int(param.otlpTLS), param.otlpSampleRatio),
	}

	args = append(args, param.protoPath...)
	args = append(args, "--go_out="+pbOutput)
	args = append(args, "--go-grpc_out="+pbOutput)
	args = append(args, fmt.Sprintf(
		"--gripmock_out=%s:%s",
		strings.Join(gmOut, ","), param.output))
	protoc := exec.Command("protoc", args...)
	protoc.Stdout = os.Stdout
	protoc.Stderr = os.Stderr
	err := protoc.Run()
	if err != nil {
		zerolog.Ctx(ctx).Fatal().Err(err).Msg("fail on protoc")
	}
}

func bool2int(b bool) int {
	if b {
		return 1
	}

	return 0
}

// append gopackage in proto files if doesn't have any.
func fixGoPackage(ctx context.Context, protoPaths []string) []string {
	results := make([]string, 0, len(protoPaths))

	for _, protoPath := range protoPaths {
		pile, err := os.OpenFile(protoPath, os.O_RDONLY, 0o600)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msgf("unable to open protofile %s", protoPath)

			continue
		}

		defer pile.Close()

		packageName := "protogen/" + strings.Trim(filepath.Dir(protoPath), "/")

		if err := os.MkdirAll(packageName, 0o666); err != nil {
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
