package sdk

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/app"
	grpcclient "github.com/bavix/gripmock/v3/internal/infra/grpcclient"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type embeddedMock struct {
	conn       *grpc.ClientConn
	server     *grpc.Server
	lis        net.Listener
	addr       string
	budgerigar *stuber.Budgerigar
	recorder   *InMemoryRecorder
}

func (m *embeddedMock) Close() error {
	if m.conn != nil {
		_ = m.conn.Close()
		m.conn = nil
	}

	if m.lis != nil {
		_ = m.lis.Close()
		m.lis = nil
	}

	if m.server != nil {
		m.server.GracefulStop()
		m.server = nil
	}

	return nil
}

//nolint:funlen
func runEmbedded(ctx context.Context, o *options) (*embeddedMock, error) {
	timeout := o.healthyTimeout
	if timeout == 0 {
		timeout = defaultHealthyTimeout
	}

	budgerigar := stuber.NewBudgerigar()
	waiter := app.NewInstantExtender()
	recorder := &InMemoryRecorder{}

	fds := &descriptorpb.FileDescriptorSet{File: o.descriptorFiles}

	server, err := app.BuildFromDescriptorSet(ctx, fds, budgerigar, waiter, recorder)
	if err != nil {
		return nil, err
	}

	listenAddr := o.listenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1:0"
	}

	if o.listenNetwork == "" {
		o.listenNetwork = "tcp"
	}

	lis, err := net.Listen(o.listenNetwork, listenAddr) //nolint:noctx
	if err != nil {
		server.GracefulStop()

		return nil, err
	}

	addr := listenAddrString(lis)

	go func() { _ = server.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///"+addr, embeddedDialOptions(o)...)
	if err != nil {
		_ = lis.Close()

		server.GracefulStop()

		return nil, err
	}

	if err := waitForHealthy(ctx, conn, timeout); err != nil {
		_ = conn.Close()
		_ = lis.Close()

		server.GracefulStop()

		return nil, err
	}

	return &embeddedMock{
		conn:       conn,
		server:     server,
		lis:        lis,
		addr:       addr,
		budgerigar: budgerigar,
		recorder:   recorder,
	}, nil
}

func embeddedDialOptions(o *options) []grpc.DialOption {
	const maxDialOptions = 3

	opts := make([]grpc.DialOption, 0, maxDialOptions)
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if o.session == "" {
		return opts
	}

	return append(opts,
		grpc.WithChainUnaryInterceptor(grpcclient.UnarySessionInterceptor(o.session)),
		grpc.WithChainStreamInterceptor(grpcclient.StreamSessionInterceptor(o.session)),
	)
}

var ErrServerNotHealthy = errors.New("gripmock: server did not become healthy")

func waitForHealthy(ctx context.Context, conn *grpc.ClientConn, timeout time.Duration) error {
	started := time.Now()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := grpc_health_v1.NewHealthClient(conn)

	ticker := time.NewTicker(50 * time.Millisecond) //nolint:mnd
	defer ticker.Stop()

	var lastCause string

	for {
		status, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: app.HealthServiceName})

		switch {
		case err != nil:
			lastCause = err.Error()
		case status.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING:
			return nil
		default:
			lastCause = "status " + status.GetStatus().String()
		}

		select {
		case <-ctx.Done():
			if lastCause == "" {
				lastCause = "no health response yet"
			}

			return fmt.Errorf("%w: %s did not report SERVING within %s (waited %s, last check: %s)",
				ErrServerNotHealthy, conn.Target(), timeout, time.Since(started).Round(time.Millisecond), lastCause)
		case <-ticker.C:
		}
	}
}

func listenAddrString(l net.Listener) string {
	if tcp, ok := l.Addr().(*net.TCPAddr); ok {
		return fmt.Sprintf("127.0.0.1:%d", tcp.Port)
	}

	return l.Addr().String()
}
