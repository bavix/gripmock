package testkit

import (
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// StartHealthGRPC starts a minimal gRPC server that serves health checks.
//
// It is intended for SDK remote-mode tests where a real GripMock process is
// not required, but Run(...WithRemote...) still needs a healthy gRPC endpoint.
func StartHealthGRPC(t testing.TB) (addr string) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	hs := health.NewServer()
	hs.SetServingStatus("gripmock", healthgrpc.HealthCheckResponse_SERVING)

	// Test-only local health server for SDK remote-mode tests.
	gs := grpc.NewServer() // nosemgrep: go.grpc.security.grpc-server-insecure-connection.grpc-server-insecure-connection
	healthgrpc.RegisterHealthServer(gs, hs)

	go func() { _ = gs.Serve(lis) }()

	t.Cleanup(func() {
		gs.Stop()
		_ = lis.Close()
	})

	return lis.Addr().String()
}
