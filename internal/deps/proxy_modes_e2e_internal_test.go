package deps

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/config"
	protodom "github.com/bavix/gripmock/v3/internal/domain/proto"
)

func startUpstream(t *testing.T, protoPath string) *e2eServer {
	t.Helper()

	return startGripmockWith(t, protoPath, nil)
}

func startGripmockWith(t *testing.T, protoPath string, args *protodom.Arguments) *e2eServer {
	t.Helper()

	cfg := config.Load()
	cfg.GRPC.Addr = freeAddr(t)
	cfg.HTTP.Addr = freeAddr(t)
	cfg.Gateway.Addr = freeAddr(t)

	builder := NewBuilder(WithConfig(cfg))

	ctx, cancel := context.WithCancel(t.Context())

	if args == nil {
		args = protodom.New([]string{protoPath}, nil, nil)
	}

	go func() {
		rest, err := builder.RestServe(ctx, "")
		if err != nil {
			return
		}

		_ = rest.ListenAndServe()
	}()
	go func() { _ = builder.GRPCServe(ctx, args) }()

	t.Cleanup(func() {
		cancel()

		stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(context.Background()), shutdownBudget)
		defer stopCancel()

		builder.Shutdown(stopCtx)
	})

	srv := &e2eServer{
		grpcAddr:  cfg.GRPC.Addr,
		restURL:   "http://" + cfg.HTTP.Addr,
		protoPath: protoPath,
	}

	waitServing(t, srv)

	return srv
}

func waitServing(t *testing.T, srv *e2eServer) {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)

	for {
		if serviceKnown(t, srv) {
			return
		}

		require.False(t, time.Now().After(deadline), "e2e.Greeter never appeared on "+srv.restURL)

		time.Sleep(50 * time.Millisecond)
	}
}

func writeE2EProto(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "e2e.proto")
	require.NoError(t, os.WriteFile(path, []byte(e2eProto), 0o600))

	return path
}

func sayHelloTo(
	t *testing.T,
	srv *e2eServer,
	in, out protoreflect.MessageDescriptor,
	name string,
) (*dynamicpb.Message, error) {
	t.Helper()

	conn, err := grpc.NewClient(srv.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString(name))

	reply := dynamicpb.NewMessage(out)
	if err := conn.Invoke(t.Context(), "/e2e.Greeter/SayHello", request, reply); err != nil {
		return nil, err
	}

	return reply, nil
}

func replyMessage(t *testing.T, reply *dynamicpb.Message, out protoreflect.MessageDescriptor) string {
	t.Helper()

	return reply.Get(out.Fields().ByName("message")).String()
}

func stubCount(t *testing.T, srv *e2eServer) int {
	t.Helper()

	var stubs []map[string]any

	require.True(t, getJSON(t, srv.restURL+"/api/stubs", &stubs))

	return len(stubs)
}

func proxyArgs(protoPath, mode, upstream string) *protodom.Arguments {
	return protodom.NewWithBindings(
		[]string{protoPath},
		nil,
		[]protodom.ProxySourceBinding{{
			ProxyURL: "grpc+" + mode + "://" + upstream,
			Sources:  []string{protoPath},
		}},
	)
}

func TestProxyModeForwardsUpstream(t *testing.T) { //nolint:paralleltest // boots two servers
	protoPath := writeE2EProto(t)
	upstream := startUpstream(t, protoPath)
	upstream.addStub(t, "Alex", "from upstream")

	in, out := compileE2EDescriptors(t, protoPath)

	front := startGripmockWith(t, protoPath, proxyArgs(protoPath, "proxy", upstream.grpcAddr))
	front.addStub(t, "Alex", "from the local stub")

	reply, err := sayHelloTo(t, front, in, out, "Alex")
	require.NoError(t, err)
	require.Equal(t, "from upstream", replyMessage(t, reply, out), "proxy mode must ignore local stubs")

	var records []map[string]any

	require.True(t, getJSON(t, front.restURL+"/api/history", &records))
	require.NotEmpty(t, records, "the proxied call must be recorded once")
}

func TestReplayModePrefersLocalStubs(t *testing.T) { //nolint:paralleltest // boots two servers
	protoPath := writeE2EProto(t)
	upstream := startUpstream(t, protoPath)
	upstream.addStub(t, "Alex", "from upstream")
	upstream.addStub(t, "OnlyUpstream", "upstream only")

	in, out := compileE2EDescriptors(t, protoPath)

	front := startGripmockWith(t, protoPath, proxyArgs(protoPath, "replay", upstream.grpcAddr))
	front.addStub(t, "Alex", "from the local stub")

	local, err := sayHelloTo(t, front, in, out, "Alex")
	require.NoError(t, err)
	require.Equal(t, "from the local stub", replyMessage(t, local, out))

	before := stubCount(t, front)

	fallback, err := sayHelloTo(t, front, in, out, "OnlyUpstream")
	require.NoError(t, err)
	require.Equal(t, "upstream only", replyMessage(t, fallback, out), "a local miss must fall back upstream")

	require.Equal(t, before, stubCount(t, front), "replay must not record the upstream answer")
}

func TestCaptureModeRecordsUpstreamMisses(t *testing.T) { //nolint:paralleltest // boots two servers
	protoPath := writeE2EProto(t)
	upstream := startUpstream(t, protoPath)
	upstream.addStub(t, "Captured", "captured reply")

	in, out := compileE2EDescriptors(t, protoPath)

	front := startGripmockWith(t, protoPath, proxyArgs(protoPath, "capture", upstream.grpcAddr))

	before := stubCount(t, front)

	reply, err := sayHelloTo(t, front, in, out, "Captured")
	require.NoError(t, err)
	require.Equal(t, "captured reply", replyMessage(t, reply, out))

	deadline := time.Now().Add(5 * time.Second)
	for stubCount(t, front) == before && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	require.Greater(t, stubCount(t, front), before, "capture must persist the upstream answer as a stub")
}

func TestProxyMissIsNotDoubleRecorded(t *testing.T) { //nolint:paralleltest // boots two servers
	protoPath := writeE2EProto(t)
	upstream := startUpstream(t, protoPath)

	in, out := compileE2EDescriptors(t, protoPath)

	front := startGripmockWith(t, protoPath, proxyArgs(protoPath, "replay", upstream.grpcAddr))

	_, err := sayHelloTo(t, front, in, out, "NobodyAnywhere")
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))

	var records []map[string]any

	require.True(t, getJSON(t, front.restURL+"/api/history", &records))
	require.Len(t, records, 1, "the proxy leg owns the record of a miss it retried")
}

func TestUpstreamRestStaysReachable(t *testing.T) { //nolint:paralleltest // boots one server
	protoPath := writeE2EProto(t)
	upstream := startUpstream(t, protoPath)

	var payload map[string]any

	require.True(t, getJSON(t, upstream.restURL+"/api/health/readiness", &payload))
	require.Equal(t, http.StatusOK, postJSON(t, upstream.restURL+"/api/stubs", []map[string]any{{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Ping"}},
		"output":  map[string]any{"data": map[string]any{"message": "pong"}},
	}}, nil))
}

func TestCaptureKeepsLargeIntegersExact(t *testing.T) { //nolint:paralleltest // boots two servers
	protoPath := writeE2EProto(t)
	upstream := startUpstream(t, protoPath)

	upstream.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Big", "count": bigCount}},
		"output":  map[string]any{"data": map[string]any{"message": "upstream big"}},
	}, "")

	in, out := compileE2EDescriptors(t, protoPath)

	front := startGripmockWith(t, protoPath, proxyArgs(protoPath, "capture", upstream.grpcAddr))

	before := stubCount(t, front)

	reply, err := callWithCount(t, front, in, out, "Big", bigCount)
	require.NoError(t, err)
	require.Equal(t, "upstream big", reply)

	deadline := time.Now().Add(5 * time.Second)
	for stubCount(t, front) == before && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	require.Greater(t, stubCount(t, front), before)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, front.restURL+"/api/stubs", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "9007199254740993")
	require.NotContains(t, string(body), "9007199254740992")
}
