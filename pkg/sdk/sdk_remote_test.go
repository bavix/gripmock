package sdk

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

func TestRunRemoteConnectionRefused(t *testing.T) {
	t.Parallel()

	// Arrange
	grpcAddr := "127.0.0.1:15999"
	restURL := "http://127.0.0.1:16000"

	// Act
	mock, err := Run(t, WithRemote(grpcAddr, restURL), WithHealthCheckTimeout(500*time.Millisecond))

	// Assert
	if err == nil {
		_ = mock.Close()
		t.Fatal("expected error when connecting to non-existent gripmock")
	}
	require.Error(t, err)
}

func TestRunRemoteWithCustomRestURL(t *testing.T) {
	t.Parallel()

	// Act
	_, err := Run(t,
		WithRemote("127.0.0.1:15998", "http://127.0.0.1:15999"),
		WithHealthCheckTimeout(200*time.Millisecond),
	)

	// Assert
	require.Error(t, err)
}

func TestRunRemoteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Remote integration test in short mode")
	}
	t.Parallel()

	ctx := t.Context()

	// Arrange
	grpcLis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	grpcPort := grpcLis.Addr().(*net.TCPAddr).Port
	_ = grpcLis.Close()

	httpLis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	httpPort := httpLis.Addr().(*net.TCPAddr).Port
	_ = httpLis.Close()

	projRoot := filepath.Join("..", "..")
	protoPath := filepath.Join(projRoot, "examples", "projects", "greeter", "service.proto")

	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("skipping: go not found in PATH: %v", err)
		return
	}

	goDir := filepath.Dir(goPath)
	if goroot := runtime.GOROOT(); goroot != "" {
		goDir = goDir + string(filepath.ListSeparator) + filepath.Join(goroot, "bin")
	}

	cmd := exec.CommandContext(ctx, goPath, "run", ".", protoPath)
	cmd.Dir = projRoot
	env := make([]string, 0, len(os.Environ())+4)
	grpcVar := "GRPC_PORT=" + fmt.Sprintf("%d", grpcPort)
	httpVar := "HTTP_PORT=" + fmt.Sprintf("%d", httpPort)
	safePath := "PATH=" + goDir
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GRPC_PORT=") || strings.HasPrefix(e, "HTTP_PORT=") || strings.HasPrefix(e, "PATH=") {
			continue
		}

		env = append(env, e)
	}

	cmd.Env = append(env, safePath, grpcVar, httpVar)
	if err := cmd.Start(); err != nil {
		t.Skipf("skipping: cannot start gripmock: %v", err)
		return
	}
	defer func() { _ = cmd.Process.Kill() }()

	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
	restURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)

	// Act
	time.Sleep(8 * time.Second)

	mock, err := Run(t,
		WithRemote(grpcAddr, restURL),
		WithHealthCheckTimeout(10*time.Second),
	)
	if err != nil {
		t.Skipf("skipping: cannot connect to gripmock: %v", err)
		return
	}

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi from Remote")).
		Commit()

	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)
	reg, err := protodesc.NewFiles(fdsSlice[0])
	require.NoError(t, err)
	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")

	// Assert
	require.Equal(t, "Hi from Remote", getMessageField(t, msg, "message"))
	require.Equal(t, 1, mock.History().Count())

	calls := mock.History().FilterByMethod("helloworld.Greeter", "SayHello")
	require.Len(t, calls, 1)
	require.Equal(t, "Alex", calls[0].Request["name"])
	mock.Verify().Method(By("/helloworld.Greeter/SayHello")).Called(t, 1)
	mock.Verify().Total(t, 1)
}
