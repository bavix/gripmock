package sdk

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

const bufconnSize = 1024 * 1024

type embeddedMock struct {
	conn       *grpc.ClientConn
	server     *grpc.Server
	lis        net.Listener
	bufLis     *bufconn.Listener
	addr       string
	budgerigar *stuber.Budgerigar
	recorder   *InMemoryRecorder
}

func (m *embeddedMock) Conn() *grpc.ClientConn { return m.conn }
func (m *embeddedMock) Addr() string           { return m.addr }
func (m *embeddedMock) Stub(service, method string) StubBuilder {
	return &stubBuilder{mock: m, service: service, method: method}
}
func (m *embeddedMock) History() HistoryReader { return m.recorder }
func (m *embeddedMock) Verify() Verifier       { return &verifier{recorder: m.recorder} }
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

func runEmbedded(ctx context.Context, o *options) (Mock, error) {
	timeout := o.healthyTimeout
	if timeout == 0 {
		timeout = defaultHealthyTimeout
	}

	budgerigar := stuber.NewBudgerigar(features.New())
	waiter := app.NewInstantExtender()
	recorder := &InMemoryRecorder{}

	server, err := app.BuildFromDescriptorSet(ctx, o.descriptors, budgerigar, waiter, recorder)
	if err != nil {
		return nil, err
	}

	var lis net.Listener
	var bufLis *bufconn.Listener
	addr := "bufnet"

	if o.listenAddr == "" {
		bufLis = bufconn.Listen(bufconnSize)
		lis = bufLis
	} else {
		network := o.listenNetwork
		if network == "" {
			network = "tcp"
		}
		var listenErr error
		lis, listenErr = net.Listen(network, o.listenAddr)
		if listenErr != nil {
			server.GracefulStop()
			return nil, listenErr
		}
		addr = listenAddrString(lis)
	}

	go func() { _ = server.Serve(lis) }()

	var conn *grpc.ClientConn
	if bufLis != nil {
		conn, err = grpc.NewClient("passthrough:///bufnet",
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return bufLis.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		conn, err = grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
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
		bufLis:     bufLis,
		addr:       addr,
		budgerigar: budgerigar,
		recorder:   recorder,
	}, nil
}

func waitForHealthy(ctx context.Context, conn *grpc.ClientConn, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := grpc_health_v1.NewHealthClient(conn)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
				Service: app.HealthServiceName,
			})
			if err != nil {
				continue
			}
			if resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
				return nil
			}
		}
	}
}

func listenAddrString(l net.Listener) string {
	if tcp, ok := l.Addr().(*net.TCPAddr); ok {
		return fmt.Sprintf("127.0.0.1:%d", tcp.Port)
	}
	return l.Addr().String()
}
