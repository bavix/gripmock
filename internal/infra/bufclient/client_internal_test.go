package bufclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	require.NotNil(t, NewClient(config.BSRProfile{}))
}

func TestClientFetchDescriptorSetSuccess(t *testing.T) {
	t.Parallel()

	var (
		auth string
		ref  string
	)

	server := newFDSServer(&auth, &ref)
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	cli := NewClient(config.BSRProfile{
		BaseURL: parsedURL,
		Timeout: 5 * time.Second,
		Token:   "token",
	})

	fds, err := cli.FetchDescriptorSet(t.Context(), "connectrpc", "eliza", "")
	require.NoError(t, err)
	require.Len(t, fds.GetFile(), 1)
	require.Equal(t, "connectrpc/eliza/v1/eliza.proto", fds.GetFile()[0].GetName())
	require.Equal(t, "Bearer token", auth)
	require.Equal(t, "main", ref)
}

func TestClientFetchDescriptorSetDefaultRef(t *testing.T) {
	t.Parallel()

	server := newFDSServer(new(string), new(string))
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	cli := NewClient(config.BSRProfile{BaseURL: parsedURL, Timeout: 5 * time.Second})

	_, err := cli.FetchDescriptorSet(t.Context(), "connectrpc", "eliza", "v1.0.0")
	require.NoError(t, err)
}

func TestClientFetchDescriptorSetErrorStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	cli := NewClient(config.BSRProfile{BaseURL: parsedURL, Timeout: 5 * time.Second})
	_, err := cli.FetchDescriptorSet(t.Context(), "connectrpc", "eliza", "main")
	require.ErrorContains(t, err, "BSR request failed")
}

func TestClientFetchDescriptorSetDefaultTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // make sure request finishes

		_, _ = w.Write([]byte(`{"fileDescriptorSet": {}}`))
	}))
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	cli := NewClient(config.BSRProfile{BaseURL: parsedURL})
	_, err := cli.FetchDescriptorSet(t.Context(), "connectrpc", "eliza", "main")
	require.NoError(t, err)
}

// --- helpers ---

func newFDSServer(auth, ref *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/buf.registry.module.v1.FileDescriptorSetService/GetFileDescriptorSet" {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		*auth = r.Header.Get("Authorization")

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return
		}

		if rr, ok := body["resourceRef"].(map[string]any); ok {
			if n, ok := rr["name"].(map[string]any); ok {
				if v, ok := n["ref"].(string); ok {
					*ref = v
				}
			}
		}

		resp, _ := json.Marshal(map[string]any{
			"fileDescriptorSet": map[string]any{
				"file": []any{map[string]any{"name": "connectrpc/eliza/v1/eliza.proto", "package": "connectrpc.eliza.v1"}},
			},
		})

		_, _ = w.Write(resp)
	}))
}
