package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	_ "github.com/bavix/gripmock/protogen"
	"github.com/bavix/gripmock/stub"
)

func main() {
	outputPointer := flag.String("o", "", "directory to output server.go. Default is $GOPATH/src/grpc/")
	grpcPort := flag.String("grpc-port", "4770", "Port of gRPC tcp server")
	grpcBindAddr := flag.String("grpc-listen", "0.0.0.0", "Adress the gRPC server will bind to. Default to localhost, set to 0.0.0.0 to use from another machine")
	adminport := flag.String("admin-port", "4771", "Port of stub admin server")
	adminBindAddr := flag.String("admin-listen", "0.0.0.0", "Adress the admin server will bind to. Default to localhost, set to 0.0.0.0 to use from another machine")
	stubPath := flag.String("stub", "", "Path where the stub files are (Optional)")
	imports := flag.String("imports", "/protobuf", "comma separated imports path. default path /protobuf is where gripmock Dockerfile install WKT protos")
	// for backwards compatibility
	if os.Args[1] == "gripmock" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	flag.Parse()
	fmt.Println("Starting GripMock")
	if os.Getenv("GOPATH") == "" {
		log.Fatal("$GOPATH is empty")
	}
	output := *outputPointer
	if output == "" {
		output = os.Getenv("GOPATH") + "/src/grpc"
	}

	// for safety
	output += "/"
	if _, err := os.Stat(output); errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(output, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}

	ctx := context.Background()

	chReady := make(chan struct{})

	// run admin stub server
	stub.RunRestServer(chReady, stub.Options{
		StubPath: *stubPath,
		Port:     *adminport,
		BindAddr: *adminBindAddr,
	})

	// parse proto files
	protoPaths := flag.Args()

	if len(protoPaths) == 0 {
		log.Fatal("Need at least one proto file")
	}

	importDirs := strings.Split(*imports, ",")

	// generate pb.go and grpc server based on proto
	generateProtoc(protocParam{
		protoPath:   protoPaths,
		adminPort:   *adminport,
		grpcAddress: *grpcBindAddr,
		grpcPort:    *grpcPort,
		output:      output,
		imports:     importDirs,
	})

	// and run
	run, runerr := runGrpcServer(output)

	// This is a kind of crutch, but now there is no other solution.
	//I have an idea to combine gripmock and grpcmock services into one, then this check will be easier to do.
	// Checking the grpc port of the service. If the port appears, the service has started successfully.
	go func() {
		var d net.Dialer

		for {
			dialCtx, cancel := context.WithTimeout(ctx, time.Second)

			conn, err := d.DialContext(dialCtx, "tcp", net.JoinHostPort(*grpcBindAddr, *grpcPort))
			cancel()

			if err == nil && conn != nil {
				chReady <- struct{}{}

				conn.Close()

				break
			}
		}
	}()

	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGTERM, syscall.SIGINT)
	select {
	case err := <-runerr:
		log.Fatal(err)
	case <-term:
		fmt.Println("Stopping gRPC Server")
		if err := run.Process.Kill(); err != nil {
			log.Fatal(err)
		}
	}
}

type protocParam struct {
	protoPath   []string
	adminPort   string
	grpcAddress string
	grpcPort    string
	output      string
	imports     []string
}

func getProtodirs(protoPath string, imports []string) []string {
	// deduced protodir from protoPath
	splitpath := strings.Split(protoPath, "/")
	protodir := ""
	if len(splitpath) > 0 {
		protodir = path.Join(splitpath[:len(splitpath)-1]...)
	}

	// search protodir prefix
	protodirIdx := -1
	for i := range imports {
		dir := path.Join("protogen", imports[i])
		if strings.HasPrefix(protodir, dir) {
			protodir = dir
			protodirIdx = i
			break
		}
	}

	protodirs := make([]string, 0, len(imports)+1)
	protodirs = append(protodirs, protodir)
	// include all dir in imports, skip if it has been added before
	for i, dir := range imports {
		if i == protodirIdx {
			continue
		}
		protodirs = append(protodirs, dir)
	}
	return protodirs
}

func generateProtoc(param protocParam) {
	param.protoPath = fixGoPackage(param.protoPath)
	protodirs := getProtodirs(param.protoPath[0], param.imports)

	// estimate args length to prevent expand
	args := make([]string, 0, len(protodirs)+len(param.protoPath)+2) //nolint:gomnd
	for _, dir := range protodirs {
		args = append(args, "-I", dir)
	}

	// the latest go-grpc plugin will generate subfolders under $GOPATH/src based on go_package option
	pbOutput := os.Getenv("GOPATH") + "/src"

	args = append(args, param.protoPath...)
	args = append(args, "--go_out="+pbOutput)
	args = append(args, "--go-grpc_out="+pbOutput)
	args = append(args, fmt.Sprintf("--gripmock_out=admin-port=%s,grpc-address=%s,grpc-port=%s:%s",
		param.adminPort, param.grpcAddress, param.grpcPort, param.output))
	protoc := exec.Command("protoc", args...)
	protoc.Stdout = os.Stdout
	protoc.Stderr = os.Stderr
	err := protoc.Run()
	if err != nil {
		log.Fatal("Fail on protoc ", err)
	}

}

// append gopackage in proto files if doesn't have any.
func fixGoPackage(protoPaths []string) []string {
	fixgopackage := exec.Command("fix_gopackage.sh", protoPaths...)
	buf := &bytes.Buffer{}
	fixgopackage.Stdout = buf
	fixgopackage.Stderr = os.Stderr
	err := fixgopackage.Run()
	if err != nil {
		log.Println("error on fixGoPackage", err)

		return protoPaths
	}

	return strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
}

func runGrpcServer(output string) (*exec.Cmd, <-chan error) {
	run := exec.Command("go", "run", output+"server.go")
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	err := run.Start()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("grpc server pid: %d\n", run.Process.Pid)
	runerr := make(chan error)
	go func() {
		runerr <- run.Wait()
	}()
	return run, runerr
}
