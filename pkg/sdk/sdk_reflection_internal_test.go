package sdk

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func TestNewServerWithReflection(t *testing.T) {
	t.Parallel()

	srv1 := mustServerWithProto(t,
		sdkProtoPath("greeter"),
		WithListenAddr("tcp", ":0"),
	)

	srv2 := NewTestServer(t,
		WithReflection(srv1.Address()),
	)

	require.Contains(t, srv2.Address(), "127.0.0.1:")
}

func TestNewServerPanicsWithoutDescriptors(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() { NewServer(t) })
}

func TestInitServerWithReflectionNoServices(t *testing.T) {
	t.Parallel()

	lc := net.ListenConfig{}
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := lis.Addr().String()

	_, port, _ := net.SplitHostPort(addr)
	addr = "127.0.0.1:" + port

	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	hs := health.NewServer()
	hs.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(server, hs)

	reflection.Register(server)
	go func() { _ = server.Serve(lis) }()

	defer server.GracefulStop()

	_, err = initServer(t, WithReflection(addr), WithHealthCheckTimeout(2*time.Second))

	require.Error(t, err)
	require.Contains(t, err.Error(), "no services found via reflection")
}

func TestInitServerWithReflectionInvalidAddr(t *testing.T) {
	t.Parallel()

	_, err := initServer(t, WithReflection("localhost:59999"), WithHealthCheckTimeout(100*time.Millisecond))

	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "failed to connect") ||
			strings.Contains(errStr, "failed to get reflection stream") ||
			strings.Contains(errStr, "connection refused"), "err=%v", err)
}
