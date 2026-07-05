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

func TestRunWithReflection(t *testing.T) {
	t.Parallel()

	mock1 := mustRunWithProto(t,
		sdkProtoPath("greeter"),
		WithListenAddr("tcp", ":0"),
		WithHealthCheckTimeout(5*time.Second), //nolint:mnd
	)

	mock2, err := Run(t,
		WithReflection(mock1.Addr()),
		WithHealthCheckTimeout(5*time.Second), //nolint:mnd
	)

	require.NoError(t, err)
	require.NotNil(t, mock2)
	require.Contains(t, mock2.Addr(), "127.0.0.1:")
}

func TestRunWithReflectionNoServices(t *testing.T) {
	t.Parallel()

	// Arrange
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

	// Act
	_, err = Run(t, WithReflection(addr), WithHealthCheckTimeout(2*time.Second)) //nolint:mnd

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "no services found via reflection")
}

func TestRunWithReflectionInvalidAddr(t *testing.T) {
	t.Parallel()

	// Act
	_, err := Run(t, WithReflection("localhost:59999"), WithHealthCheckTimeout(100*time.Millisecond)) //nolint:mnd

	// Assert
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "failed to connect") ||
			strings.Contains(errStr, "failed to get reflection stream") ||
			strings.Contains(errStr, "connection refused"), "err=%v", err)
}
