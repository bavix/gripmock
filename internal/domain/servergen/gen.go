package servergen

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/internal/domain/proto"
	"github.com/bavix/gripmock/internal/infra/patcher"
)

var errProtoNotFound = errors.New("proto not found")

// ServerGenerate is a function that runs the protoc command with the given
// parameters.
//
// It takes a context.Context and a protocParam struct as parameters. The
// protocParam struct contains the protoPath, output, and imports fields that
// are used to configure the protoc command.
//
// It generates the protoc command arguments and runs the protoc command with
// the given parameters. If there is an error running the protoc command, it
// logs a fatal message.
func ServerGenerate(ctx context.Context, param *proto.ProtocParam) error {
	// Check if output folder exists, if not create it
	// nosemgrep:semgrep-go.os-error-is-not-exist
	if _, err := os.Stat(param.Output()); os.IsNotExist(err) {
		// Create output folder
		if err := os.Mkdir(param.Output(), os.ModePerm); err != nil {
			zerolog.Ctx(ctx).Fatal().Err(err).Msg("unable to create output folder")
		}
	}

	// Fix the go_package option for each proto file in the protoPath.
	protoPath := fixGoPackage(ctx, param.ProtoPath())
	if len(protoPath) == 0 {
		return errProtoNotFound
	}

	// Get the proto directories based on the protoPath and imports.
	protoDirs := getProtodirs(ctx, protoPath[0], param.Imports())

	// Estimate the length of the args slice to prevent expanding it.
	args := make([]string, 0, len(protoDirs)+len(protoPath)+2) //nolint:mnd

	// Append the -I option for each proto directory to the args slice.
	for _, dir := range protoDirs {
		args = append(args, "-I", dir)
	}

	// Set the output directory for generated files to $GOPATH/src.
	pbOutput := os.Getenv("GOPATH") + "/src"

	// Append the protoPath, --go_out, --go-grpc_out, and --gripmock_out options
	// to the args slice.
	args = append(args, protoPath...)
	args = append(args, "--go_out="+pbOutput)
	args = append(args, "--go-grpc_out="+pbOutput)
	args = append(args, "--gripmock_out="+param.Output())

	// Create a new exec.Cmd command with the protoc command and the args.
	protoc := exec.Command("protoc", args...)

	// Set the environment variables for the command.
	protoc.Env = os.Environ()

	// Set the stdout and stderr for the command.
	protoc.Stdout = os.Stdout
	protoc.Stderr = os.Stderr

	return protoc.Run()
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
