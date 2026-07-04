package sdk

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

const bufconnSize = 1024 * 1024

type embeddedMock struct {
	conn          *grpc.ClientConn
	server        *grpc.Server
	lis           net.Listener
	bufLis        *bufconn.Listener
	addr          string
	budgerigar    *stuber.Budgerigar
	recorder      *InMemoryRecorder
	expectedTotal atomic.Int32
	expectedMu    sync.Mutex
	expectedByMth map[string]int
}

func (m *embeddedMock) Conn() *grpc.ClientConn { return m.conn }
func (m *embeddedMock) Addr() string           { return m.addr }
func (m *embeddedMock) Stub(service, method string) StubBuilder {
	if strings.TrimSpace(service) == "" || strings.TrimSpace(method) == "" {
		panic("sdk.Mock.Stub: service and method must be non-empty")
	}

	return &stubBuilderCore{
		service: service,
		method:  method,
		onCommit: func(stub *stuber.Stub) error {
			return m.commitStubs([]*stuber.Stub{stub})
		},
	}
}
func (m *embeddedMock) History() HistoryReader { return m.recorder }
func (m *embeddedMock) Verify() Verifier {
	return &verifier{recorder: m.recorder, expectedTotal: &m.expectedTotal, expectedByMth: m.expectedByMth, expectedMu: &m.expectedMu}
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

func (m *embeddedMock) commitStubs(stubs []*stuber.Stub) error {
	for _, stub := range stubs {
		m.budgerigar.PutMany(stub)
		if stub.Options.Times > 0 {
			m.expectedTotal.Add(int32(stub.Options.Times)) //nolint:gosec

			m.expectedMu.Lock()
			if m.expectedByMth == nil {
				m.expectedByMth = make(map[string]int)
			}
			m.expectedByMth[methodKey(stub.Service, stub.Method)] += stub.Options.Times
			m.expectedMu.Unlock()
		}
	}

	return nil
}

func runEmbedded(ctx context.Context, o *options) (Mock, error) {
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

	lis, bufLis, addr, err := setupEmbeddedListener(o, server)
	if err != nil {
		server.GracefulStop()

		return nil, err
	}

	go func() { _ = server.Serve(lis) }()

	conn, err := setupEmbeddedClient(addr, bufLis)
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

func setupEmbeddedListener(o *options, server *grpc.Server) (net.Listener, *bufconn.Listener, string, error) {
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
		var lc net.ListenConfig
		var listenErr error
		lis, listenErr = lc.Listen(context.Background(), network, o.listenAddr)
		if listenErr != nil {
			return nil, nil, "", listenErr
		}
		addr = listenAddrString(lis)
	}

	return lis, bufLis, addr, nil
}

func setupEmbeddedClient(addr string, bufLis *bufconn.Listener) (*grpc.ClientConn, error) {
	if bufLis != nil {
		return grpc.NewClient("passthrough:///bufnet",
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return bufLis.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
