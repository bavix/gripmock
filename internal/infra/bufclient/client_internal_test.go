package bufclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	value := NewClient(config.Config{BSRTimeout: 5 * time.Second})
	require.NotNil(t, value)
}

func TestFetchDescriptorSetSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fdsEndpoint {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		response := map[string]any{
			"fileDescriptorSet": map[string]any{
				"file": []any{
					map[string]any{"name": "connectrpc/eliza/v1/eliza.proto", "package": "connectrpc.eliza.v1"},
				},
			},
		}

		payload, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cli := NewClient(config.Config{BSRBaseURL: server.URL, BSRTimeout: 5 * time.Second})

	fds, err := cli.FetchDescriptorSet(t.Context(), "buf.build/connectrpc/eliza", "main")
	require.NoError(t, err)
	require.Len(t, fds.GetFile(), 1)
	require.Equal(t, "connectrpc/eliza/v1/eliza.proto", fds.GetFile()[0].GetName())
}

func TestFetchDescriptorSetErrorStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	defer server.Close()

	cli := NewClient(config.Config{BSRBaseURL: server.URL, BSRTimeout: 5 * time.Second})
	_, err := cli.FetchDescriptorSet(t.Context(), "buf.build/connectrpc/eliza", "main")
	require.Error(t, err)
	require.Contains(t, err.Error(), "BSR request failed")
}

func TestFetchDescriptorSetInvalidModule(t *testing.T) {
	t.Parallel()

	cli := NewClient(config.Config{BSRTimeout: 5 * time.Second})
	_, err := cli.FetchDescriptorSet(t.Context(), "broken", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid BSR module")
}
