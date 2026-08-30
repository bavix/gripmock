package deps

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/config"
	protodom "github.com/bavix/gripmock/v3/internal/domain/proto"
	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

func selfSignedCert(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()

	dir := t.TempDir()
	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	derKey, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	derCert, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(certFile,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCert}), 0o600))
	require.NoError(t, os.WriteFile(keyFile,
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derKey}), 0o600))

	parsed, err := x509.ParseCertificate(derCert)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(parsed)

	return certFile, keyFile, pool
}

func localhostAddr(addr string) string {
	return strings.Replace(addr, "127.0.0.1", "localhost", 1)
}

func TestTLSAcrossEveryServer(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	certFile, keyFile, pool := selfSignedCert(t)

	cfg := config.Load()

	addrs := reserveAddrs(t, 3)
	cfg.GRPC.Addr, cfg.HTTP.Addr, cfg.Gateway.Addr = addrs[0], addrs[1], addrs[2]
	cfg.GRPCTLS.CertFile = certFile
	cfg.GRPCTLS.KeyFile = keyFile
	cfg.HTTPTLS.CertFile = certFile
	cfg.HTTPTLS.KeyFile = keyFile
	cfg.GatewayTLS.CertFile = certFile
	cfg.GatewayTLS.KeyFile = keyFile

	builder := NewBuilder(WithConfig(cfg))

	ctx, cancel := context.WithCancel(t.Context())

	bootErr := make(chan error, 3)

	go func() {
		rest, err := builder.RestServe(ctx, "")
		if err != nil {
			bootErr <- err

			return
		}

		bootErr <- rest.ListenAndServe()
	}()
	go func() { bootErr <- builder.GatewayServe(ctx) }()
	go func() { bootErr <- builder.GRPCServe(ctx, protodom.New([]string{protoPath}, nil, nil)) }()

	t.Cleanup(func() {
		cancel()

		stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(context.Background()), shutdownBudget)
		defer stopCancel()

		builder.Shutdown(stopCtx)
	})

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}}}

	restURL := "https://" + localhostAddr(cfg.HTTP.Addr)
	waitTLSReady(t, client, restURL)
	addSecureStub(t, client, restURL)

	in, out := compileE2EDescriptors(t, protoPath)
	callSecureGRPC(t, localhostAddr(cfg.GRPC.Addr), pool, in, out)
	callSecureGateway(t, client, localhostAddr(cfg.Gateway.Addr))
	rejectPlaintext(t, cfg.Gateway.Addr)
}

func addSecureStub(t *testing.T, client *http.Client, restURL string) {
	t.Helper()

	body := strings.NewReader(`[{
		"service": "e2e.Greeter",
		"method": "SayHello",
		"input": {"equals": {"name": "Secure"}},
		"output": {"data": {"message": "over tls"}}
	}]`)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, restURL+"/api/stubs", body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func callSecureGRPC(t *testing.T, addr string, pool *x509.CertPool, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(pool, "")))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Secure"))

	reply := dynamicpb.NewMessage(out)
	require.NoError(t, conn.Invoke(t.Context(), "/e2e.Greeter/SayHello", request, reply))
	require.Equal(t, "over tls", reply.Get(out.Fields().ByName("message")).String())
}

func callSecureGateway(t *testing.T, client *http.Client, addr string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		"https://"+addr+"/e2e.Greeter/SayHello", strings.NewReader(`{"name":"Secure"}`))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	require.JSONEq(t, `{"message":"over tls"}`, string(body))
}

func rejectPlaintext(t *testing.T, addr string) {
	t.Helper()

	resp, err := (&http.Client{}).Get("http://" + addr + "/e2e.Greeter/SayHello") //nolint:noctx
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Contains(t, string(body), "HTTP request to an HTTPS server")
}

func waitTLSReady(t *testing.T, client *http.Client, restURL string) {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)

	for {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, restURL+"/api/services", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		if err == nil {
			var services []struct {
				ID string `json:"id"`
			}

			decodeErr := json.NewDecoder(resp.Body).Decode(&services)
			require.NoError(t, resp.Body.Close())

			if decodeErr == nil {
				for _, service := range services {
					if service.ID == "e2e.Greeter" {
						return
					}
				}
			}
		}

		require.False(t, time.Now().After(deadline), "the TLS servers never became ready")

		time.Sleep(50 * time.Millisecond)
	}
}

func TestStubsAreLoadedFromDiskAtStartup(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)

	stubDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(stubDir, "hello.json"), []byte(`{
		"service": "e2e.Greeter",
		"method": "SayHello",
		"input": {"equals": {"name": "FromDisk"}},
		"output": {"data": {"message": "loaded from disk"}}
	}`), 0o600))

	cfg := config.Load()

	addrs := reserveAddrs(t, 3)
	cfg.GRPC.Addr, cfg.HTTP.Addr, cfg.Gateway.Addr = addrs[0], addrs[1], addrs[2]

	builder := NewBuilder(WithConfig(cfg))

	ctx, cancel := context.WithCancel(t.Context())

	bootErr := make(chan error, 3)

	go func() {
		rest, err := builder.RestServe(ctx, stubDir)
		if err != nil {
			bootErr <- err

			return
		}

		bootErr <- rest.ListenAndServe()
	}()
	go func() { bootErr <- builder.GRPCServe(ctx, protodom.New([]string{protoPath}, nil, nil)) }()

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

	waitServing(t, srv, bootErr)

	in, out := compileE2EDescriptors(t, protoPath)

	reply, err := sayHelloTo(t, srv, in, out, "FromDisk")
	require.NoError(t, err)
	require.Equal(t, "loaded from disk", replyMessage(t, reply, out))

	require.Equal(t, 1, stubCount(t, srv))
}

func TestTLSMinVersionIsEnforcedOnTheWire(t *testing.T) { //nolint:paralleltest // boots a real server
	certFile, keyFile, pool := selfSignedCert(t)

	cfg := config.Load()

	httpAddrs := reserveAddrs(t, 1)
	cfg.HTTP.Addr = httpAddrs[0]
	cfg.HTTPTLS.CertFile = certFile
	cfg.HTTPTLS.KeyFile = keyFile
	cfg.HTTPTLS.MinVersion = infraTLS.MinTLSVersion13

	builder := NewBuilder(WithConfig(cfg))

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		rest, err := builder.RestServe(ctx, "")
		if err != nil {
			return
		}

		_ = rest.ListenAndServe()
	}()

	t.Cleanup(func() {
		cancel()

		stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(context.Background()), shutdownBudget)
		defer stopCancel()

		builder.Shutdown(stopCtx)
	})

	addr := localhostAddr(cfg.HTTP.Addr)
	waitTLS13Listener(t, addr, pool)

	//nolint:gosec // capping the client at 1.2 is what the assertion below is about
	dialer := &tls.Dialer{Config: &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
	}}

	refused, err := dialer.DialContext(t.Context(), "tcp", addr)
	if err == nil {
		_ = refused.Close()
	}

	require.Error(t, err,
		"the server was configured for 1.3, so a client capped at 1.2 must not complete the handshake")
}

func waitTLS13Listener(t *testing.T, addr string, pool *x509.CertPool) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for {
		dialer := &tls.Dialer{Config: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13}}

		conn, err := dialer.DialContext(t.Context(), "tcp", addr)
		if err == nil {
			require.NoError(t, conn.Close())

			return
		}

		require.False(t, time.Now().After(deadline), "the TLS 1.3 listener never came up: %v", err)

		time.Sleep(50 * time.Millisecond)
	}
}
