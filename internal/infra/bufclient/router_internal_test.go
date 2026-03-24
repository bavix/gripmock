package bufclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	require.NotNil(t, NewRouter(config.BSRConfig{}))
}

func TestRouterRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		moduleFn func(selfHost string) string
		version  string
		wantBuf  int
		wantSelf int
	}{
		{
			name: "routes to self by host",
			moduleFn: func(selfHost string) string {
				return selfHost + "/connectrpc/eliza"
			},
			wantBuf:  0,
			wantSelf: 1,
		},
		{
			name: "routes to self with explicit version",
			moduleFn: func(selfHost string) string {
				return selfHost + "/acme/payments"
			},
			version:  "main",
			wantBuf:  0,
			wantSelf: 1,
		},
		{
			name: "routes to buf when host does not match",
			moduleFn: func(string) string {
				return "unknown.host/acme/payments"
			},
			version:  "main",
			wantBuf:  1,
			wantSelf: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router, bufProbe, selfProbe, selfHost := newRouterWithProbes(t)

			fds, err := router.FetchDescriptorSet(t.Context(), tt.moduleFn(selfHost), tt.version)
			require.NoError(t, err)
			require.Len(t, fds.GetFile(), 1)

			requireProbeCounts(t, bufProbe, selfProbe, tt.wantBuf, tt.wantSelf)
		})
	}
}

func TestRouterInvalidModule(t *testing.T) {
	t.Parallel()

	router := NewRouter(config.BSRConfig{})

	for _, module := range []string{"broken", "buf.build/connectrpc/eliza/extra"} {
		_, err := router.FetchDescriptorSet(t.Context(), module, "")
		require.ErrorContains(t, err, "invalid BSR module")
	}
}

func TestRouterWithScheme(t *testing.T) {
	t.Parallel()

	server := newFDSServer(new(string), new(string))
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	router := NewRouter(config.BSRConfig{
		Buf: config.BSRProfile{BaseURL: parsedURL, Timeout: 5 * time.Second},
	})

	_, err := router.FetchDescriptorSet(t.Context(), "https://buf.build/connectrpc/eliza", "main")
	require.NoError(t, err)
}

func TestParseModulePreservesPort(t *testing.T) {
	t.Parallel()

	remote, owner, repo, err := parseModule("https://self.local:8443/acme/payments")
	require.NoError(t, err)
	require.Equal(t, "self.local:8443", remote)
	require.Equal(t, "acme", owner)
	require.Equal(t, "payments", repo)
}

// --- helpers ---

type probe struct {
	mu    sync.Mutex
	calls int
	auth  string
}

func (p *probe) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.calls
}

func newProbeServer(p *probe) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/buf.registry.module.v1.FileDescriptorSetService/GetFileDescriptorSet" {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		p.mu.Lock()
		p.calls++
		p.auth = r.Header.Get("Authorization")
		p.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"fileDescriptorSet": {"file": [{"name": "service.proto", "package": "acme.v1"}]}}`))
	}))
}

func newRouterWithProbes(t *testing.T) (*Router, *probe, *probe, string) {
	t.Helper()

	bufProbe := &probe{}
	selfProbe := &probe{}

	bufServer := newProbeServer(bufProbe)
	t.Cleanup(bufServer.Close)

	selfServer := newProbeServer(selfProbe)
	t.Cleanup(selfServer.Close)

	bufURL, err := url.Parse(bufServer.URL)
	require.NoError(t, err)

	selfURL, err := url.Parse(selfServer.URL)
	require.NoError(t, err)

	router := NewRouter(config.BSRConfig{
		Buf:  config.BSRProfile{BaseURL: bufURL, Token: "buf-token", Timeout: 5 * time.Second},
		Self: config.BSRProfile{BaseURL: selfURL, Token: "self-token", Timeout: 5 * time.Second},
	})

	return router, bufProbe, selfProbe, selfURL.Host
}

func requireProbeCounts(t *testing.T, bufProbe, selfProbe *probe, wantBuf, wantSelf int) {
	t.Helper()
	require.Equal(t, wantBuf, bufProbe.count())
	require.Equal(t, wantSelf, selfProbe.count())
}
