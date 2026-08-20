package deps

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
	protodom "github.com/bavix/gripmock/v3/internal/domain/proto"
)

type e2eOptions struct {
	stubPath string
	mutate   func(*config.Config)
}

func startConfigured(t *testing.T, protoPath string, opts e2eOptions) *e2eServer {
	t.Helper()

	cfg := config.Load()

	addrs, releaseAddrs := reserveAddrs(t, 3)
	cfg.GRPC.Addr, cfg.HTTP.Addr, cfg.Gateway.Addr = addrs[0], addrs[1], addrs[2]

	if opts.mutate != nil {
		opts.mutate(&cfg)
	}

	builder := NewBuilder(WithConfig(cfg))

	ctx, cancel := context.WithCancel(t.Context())

	bootErr := make(chan error, 3)

	go func() {
		rest, err := builder.RestServe(ctx, opts.stubPath)
		if err != nil {
			bootErr <- err

			return
		}

		bootErr <- rest.ListenAndServe()
	}()
	go func() { bootErr <- builder.GatewayServe(ctx) }()
	go func() { bootErr <- builder.GRPCServe(ctx, protodom.New([]string{protoPath}, nil, nil)) }()

	releaseAddrs()

	t.Cleanup(func() {
		cancel()

		stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(context.Background()), shutdownBudget)
		defer stopCancel()

		builder.Shutdown(stopCtx)
	})

	srv := &e2eServer{
		grpcAddr:    cfg.GRPC.Addr,
		restURL:     "http://" + cfg.HTTP.Addr,
		gatewayAddr: cfg.Gateway.Addr,
		protoPath:   protoPath,
	}

	waitServing(t, srv, bootErr)

	return srv
}

func TestSessionGCDropsExpiredSessions(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)

	srv := startConfigured(t, protoPath, e2eOptions{mutate: func(cfg *config.Config) {
		cfg.SessionGCInterval = 100 * time.Millisecond
		cfg.SessionGCTTL = 300 * time.Millisecond
	}})

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Ephemeral"}},
		"output":  map[string]any{"data": map[string]any{"message": "bye soon"}},
	}, "short-lived")

	in, out := compileE2EDescriptors(t, protoPath)

	resp := connectJSON(t, srv, "SayHello", `{"name":"Ephemeral"}`,
		[2]string{"X-Gripmock-Session", "short-lived"})
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))

	deadline := time.Now().Add(10 * time.Second)
	for stubCount(t, srv) > 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	require.Zero(t, stubCount(t, srv), "the GC must delete the stubs of an expired session")

	var records []map[string]any

	require.True(t, getJSON(t, srv.restURL+"/api/history", &records))
	require.Empty(t, records, "the GC must delete the history of an expired session")

	_, err := sayHelloTo(t, srv, in, out, "Ephemeral")
	require.Error(t, err)
}

func TestHistoryRedactsAndTruncates(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)

	const maxMessageBytes = 64

	srv := startConfigured(t, protoPath, e2eOptions{mutate: func(cfg *config.Config) {
		cfg.HistoryRedactKeys = []string{"name"}
		cfg.HistoryMessageMaxBytes = maxMessageBytes
	}})

	srv.addStub(t, "Alex", "hello")
	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": strings.Repeat("L", 512)}},
		"output":  map[string]any{"data": map[string]any{"message": strings.Repeat("R", 512)}},
	}, "")

	in, out := compileE2EDescriptors(t, protoPath)

	_, err := sayHelloTo(t, srv, in, out, "Alex")
	require.NoError(t, err)

	_, err = sayHelloTo(t, srv, in, out, strings.Repeat("L", 512))
	require.NoError(t, err)

	var records []map[string]any

	require.True(t, getJSON(t, srv.restURL+"/api/history", &records))
	require.Len(t, records, 2)

	for _, record := range records {
		encoded := recordJSON(t, record)
		require.NotContains(t, encoded, "Alex", "a redacted key must not leak the request value")
		require.NotContains(t, encoded, strings.Repeat("L", 128), "a redacted key must not leak a long value either")
		require.NotContains(t, encoded, strings.Repeat("R", 256), "long payloads must be truncated")
		require.Contains(t, encoded, "REDACTED")
	}
}

func TestConnectProtocolVersionCanBeRequired(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)

	srv := startConfigured(t, protoPath, e2eOptions{mutate: func(cfg *config.Config) {
		cfg.ConnectRequireProtocolVersion = true
	}})

	srv.addStub(t, "Alex", "Hello Alex")

	withVersion := connectJSON(t, srv, "SayHello", `{"name":"Alex"}`)
	require.Equal(t, http.StatusOK, withVersion.status, string(withVersion.body))

	without := gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/SayHello",
		"application/json", `{"name":"Alex"}`)
	require.Equal(t, http.StatusBadRequest, without.status, string(without.body))
}

func TestUIDeepLinksFallBackToTheSPA(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	srv := startConfigured(t, protoPath, e2eOptions{})

	for _, path := range []string{"/", "/stubs", "/history", "/stubs/11111111-1111-1111-1111-111111111111"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.restURL+path, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equalf(t, http.StatusOK, resp.StatusCode, "deep link %s must reach the SPA", path)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.restURL+"/api/nope", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.NotEqual(t, http.StatusOK, resp.StatusCode, "an unknown API route must not fall back to the SPA")
}

func recordJSON(t *testing.T, record map[string]any) string {
	t.Helper()

	encoded, err := json.Marshal(record)
	require.NoError(t, err)

	return string(encoded)
}
