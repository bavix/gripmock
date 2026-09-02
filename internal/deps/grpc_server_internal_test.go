package deps

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

const hangingMethod = "/deps.Hanging/Hang"

func startHangingServer(t *testing.T) (*grpc.Server, string, <-chan struct{}) {
	t.Helper()

	started := make(chan struct{})
	release := make(chan struct{})

	t.Cleanup(func() { close(release) })

	server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		close(started)

		select {
		case <-release:
		case <-stream.Context().Done():
		}

		return nil
	}))

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = server.Serve(listener)
	}()

	return server, listener.Addr().String(), started
}

func dialHangingStream(t *testing.T, addr string, started <-chan struct{}) {
	t.Helper()

	conn, err := grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	stream, err := conn.NewStream(
		t.Context(),
		&grpc.StreamDesc{StreamName: "Hang", ServerStreams: true, ClientStreams: true},
		hangingMethod,
	)
	require.NoError(t, err)
	require.NoError(t, stream.SendMsg(&emptypb.Empty{}))

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never started")
	}
}

func TestStopGRPCServerForcesStopWhenContextExpires(t *testing.T) {
	t.Parallel()

	server, addr, started := startHangingServer(t)
	dialHangingStream(t, addr, started)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)

		stopGRPCServer(ctx, server)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("stopGRPCServer never returned: graceful stop is unbounded")
	}
}

func TestStopGRPCServerReturnsBeforeDeadlineWhenIdle(t *testing.T) {
	t.Parallel()

	server, _, _ := startHangingServer(t)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()

	stopGRPCServer(ctx, server)

	require.Less(t, time.Since(start), 5*time.Second)
}
