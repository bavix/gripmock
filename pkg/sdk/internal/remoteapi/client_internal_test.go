package remoteapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type errRoundTripper struct{}

func (errRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport failed") //nolint:err113
}

func newTestServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()

	// Wrap with gzip decompression middleware to match real server behavior
	h := httputil.GzipRequestMiddleware(http.HandlerFunc(handler))

	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	return ts
}

func newTestClient(ts *httptest.Server, session string) Client {
	return Client{BaseURL: ts.URL, HTTPClient: ts.Client(), Session: session}
}

func TestClientAddStub(t *testing.T) {
	t.Parallel()

	// Arrange
	var called bool

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stubs" {
			t.Errorf("expected /api/stubs, got %s", r.URL.Path)

			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)

			return
		}

		if r.Header.Get("X-Gripmock-Session") != "A" {
			t.Error("expected session A")

			return
		}

		var payload []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)

			return
		}

		if len(payload) != 1 {
			t.Error("expected 1 stub")

			return
		}

		called = true

		w.WriteHeader(http.StatusOK)
	})

	c := newTestClient(ts, "A")

	// Act
	err := c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})

	// Assert
	require.NoError(t, err)
	require.True(t, called)
}

func TestClientBatchDeleteAcceptsNotFoundOrGone(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		statusCode int
	}{
		{name: "not-found", statusCode: http.StatusNotFound},
		{name: "gone", statusCode: http.StatusGone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/stubs/batchDelete" {
					t.Errorf("expected /api/stubs/batchDelete, got %s", r.URL.Path)

					return
				}

				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)

					return
				}

				w.WriteHeader(tc.statusCode)
			})
			c := newTestClient(ts, "")

			// Act
			err := c.BatchDelete([]uuid.UUID{uuid.New()})

			// Assert
			require.NoError(t, err)
		})
	}
}

//nolint:funlen
func TestClientUploadDescriptors(t *testing.T) {
	t.Parallel()

	// Arrange
	name := "svc.proto"

	var called bool

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/descriptors" {
			t.Errorf("expected /api/descriptors, got %s", r.URL.Path)

			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)

			return
		}

		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("expected Content-Type application/octet-stream, got %s", r.Header.Get("Content-Type"))

			return
		}

		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("unexpected error: %v", err)

			return
		}

		var fds descriptorpb.FileDescriptorSet
		if err := proto.Unmarshal(raw, &fds); err != nil {
			t.Errorf("unexpected error: %v", err)

			return
		}

		if len(fds.GetFile()) != 1 {
			t.Errorf("expected 1 file, got %d", len(fds.GetFile()))

			return
		}

		if fds.GetFile()[0].GetName() != name {
			t.Errorf("expected name %s, got %s", name, fds.GetFile()[0].GetName())

			return
		}

		called = true

		w.WriteHeader(http.StatusOK)
	})

	c := newTestClient(ts, "")

	// Act
	err := c.UploadDescriptors([]*descriptorpb.FileDescriptorProto{{Name: &name}})

	// Assert
	require.NoError(t, err)
	require.True(t, called)
}

func TestClientFetchHistory(t *testing.T) {
	t.Parallel()

	// Arrange
	now := "2026-03-29T10:00:00Z"
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/history" {
			t.Errorf("expected /api/history, got %s", r.URL.Path)

			return
		}

		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)

			return
		}

		if r.Header.Get("X-Gripmock-Session") != "A" {
			t.Error("expected session A")

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[{\"service\":\"svc\",\"method\":\"M\",\"request\":{\"x\":1},\"response\":{\"ok\":true},\"stubId\":\"550e8400-e29b-41d4-a716-446655440000\",\"timestamp\":\"" + now + "\"}]")) //nolint:lll
	})

	c := newTestClient(ts, "A")

	// Act
	history, err := c.FetchHistory()

	// Assert
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "svc", history[0].Service)
	require.Equal(t, "M", history[0].Method)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", history[0].StubID.String())
	require.InDelta(t, float64(1), history[0].Request["x"], 0.001)
}

func TestClientVerifyMethodCalledBadRequest(t *testing.T) {
	t.Parallel()

	// Arrange
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/verify" {
			t.Errorf("expected /api/verify, got %s", r.URL.Path)

			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)

			return
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad count"}`))
	})

	c := newTestClient(ts, "")

	// Act
	err := c.VerifyMethodCalled("svc", "M", 2)

	// Assert
	require.Error(t, err)

	var badReq VerifyBadRequestError
	require.ErrorAs(t, err, &badReq)
	require.Equal(t, "bad count", badReq.Error())
}

func TestClientAddStubUsesDefaultHTTPClientWhenNil(t *testing.T) {
	t.Parallel()

	// Arrange
	var called bool

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stubs" {
			t.Errorf("expected /api/stubs, got %s", r.URL.Path)

			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)

			return
		}

		called = true

		w.WriteHeader(http.StatusOK)
	})

	c := Client{BaseURL: ts.URL}

	// Act
	err := c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})

	// Assert
	require.NoError(t, err)
	require.True(t, called)
}

func TestClientAddStubsBatchPayload(t *testing.T) {
	t.Parallel()

	var gotLen int

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stubs" {
			t.Errorf("expected /api/stubs, got %s", r.URL.Path)

			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)

			return
		}

		var payload []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)

			return
		}

		gotLen = len(payload)

		w.WriteHeader(http.StatusOK)
	})

	c := newTestClient(ts, "")
	err := c.AddStubs([]*stuber.Stub{
		{ID: uuid.New(), Service: "svc", Method: "M1", Output: stuber.Output{Data: map[string]any{"ok": true}}},
		{ID: uuid.New(), Service: "svc", Method: "M2", Output: stuber.Output{Data: map[string]any{"ok": true}}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, gotLen)
}

func TestPtrOrZero(t *testing.T) {
	t.Parallel()

	// Arrange
	value := "x"
	now := time.Now().UTC()
	m := map[string]any{"k": "v"}

	// Act + Assert
	require.Empty(t, ptrOrZero[string](nil))
	require.Equal(t, value, ptrOrZero(&value))
	require.Nil(t, ptrOrZero[map[string]any](nil))
	require.Equal(t, m, ptrOrZero(&m))
	require.True(t, ptrOrZero[time.Time](nil).IsZero())
	require.Equal(t, now, ptrOrZero(&now))
}

func TestVerifyBadRequestErrorDefaultMessage(t *testing.T) {
	t.Parallel()

	err := VerifyBadRequestError{}
	require.Equal(t, "verification failed", err.Error())
}

func TestClientAddStubErrorStatus(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stubs" {
			t.Errorf("expected /api/stubs, got %s", r.URL.Path)

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	})

	c := newTestClient(ts, "")
	err := c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "add stubs failed with status 500")
}

func TestClientAddStubMarshalError(t *testing.T) {
	t.Parallel()

	c := Client{BaseURL: "http://127.0.0.1"}
	err := c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"bad": make(chan int)}}}) //nolint:lll
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to marshal stubs")
}

func TestClientBatchDeleteErrorStatus(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stubs/batchDelete" {
			t.Errorf("expected /api/stubs/batchDelete, got %s", r.URL.Path)

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	})

	c := newTestClient(ts, "")
	err := c.BatchDelete([]uuid.UUID{uuid.New()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "batch delete stubs failed with status 500")
}

func TestClientUploadDescriptorsBranches(t *testing.T) {
	t.Parallel()

	t.Run("empty-files-no-request", func(t *testing.T) {
		t.Parallel()

		var called bool

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true

			w.WriteHeader(http.StatusOK)
		})

		c := newTestClient(ts, "")
		err := c.UploadDescriptors(nil)
		require.NoError(t, err)
		require.False(t, called)
	})

	t.Run("server-error", func(t *testing.T) {
		t.Parallel()

		name := "svc.proto"
		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/descriptors" {
				t.Errorf("expected /api/descriptors, got %s", r.URL.Path)

				return
			}

			w.WriteHeader(http.StatusInternalServerError)
		})

		c := newTestClient(ts, "")
		err := c.UploadDescriptors([]*descriptorpb.FileDescriptorProto{{Name: &name}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "upload descriptors failed with status 500")
	})
}

//nolint:funlen
func TestClientFetchHistoryErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("server-status", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/history" {
				t.Errorf("expected /api/history, got %s", r.URL.Path)

				return
			}

			w.WriteHeader(http.StatusInternalServerError)
		})

		c := newTestClient(ts, "")
		_, err := c.FetchHistory()
		require.Error(t, err)
		require.Contains(t, err.Error(), "fetch history failed with status 500")
	})

	t.Run("decode-error", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/history" {
				t.Errorf("expected /api/history, got %s", r.URL.Path)

				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invalid":`))
		})

		c := newTestClient(ts, "")
		_, err := c.FetchHistory()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode history")
	})

	t.Run("nil-fields", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/history" {
				t.Errorf("expected /api/history, got %s", r.URL.Path)

				return
			}

			w.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(w).Encode([]map[string]any{{}}); err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}
		})

		c := newTestClient(ts, "")
		history, err := c.FetchHistory()
		require.NoError(t, err)
		require.Len(t, history, 1)
		require.Empty(t, history[0].Service)
		require.Empty(t, history[0].Method)
		require.Nil(t, history[0].Request)
		require.Nil(t, history[0].Response)
		require.True(t, history[0].Timestamp.IsZero())
	})
}

func TestClientVerifyMethodCalledBranches(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/verify" {
				t.Errorf("expected /api/verify, got %s", r.URL.Path)

				return
			}

			w.WriteHeader(http.StatusOK)
		})

		c := newTestClient(ts, "")
		require.NoError(t, c.VerifyMethodCalled("svc", "M", 1))
	})

	t.Run("bad-request-invalid-json", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/verify" {
				t.Errorf("expected /api/verify, got %s", r.URL.Path)

				return
			}

			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("not-json"))
		})

		c := newTestClient(ts, "")
		err := c.VerifyMethodCalled("svc", "M", 1)

		var badReq VerifyBadRequestError
		require.ErrorAs(t, err, &badReq)
		require.Equal(t, "verification failed", badReq.Error())
	})

	t.Run("server-status", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/verify" {
				t.Errorf("expected /api/verify, got %s", r.URL.Path)

				return
			}

			w.WriteHeader(http.StatusInternalServerError)
		})

		c := newTestClient(ts, "")
		err := c.VerifyMethodCalled("svc", "M", 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "verify request failed with status 500")
	})
}

func TestClientRequestErrors(t *testing.T) {
	t.Parallel()

	name := "svc.proto"

	errClient := Client{HTTPClient: &http.Client{Transport: errRoundTripper{}}}

	for _, group := range []struct {
		name   string
		client Client
		err    string
	}{
		{name: "url-build", client: Client{BaseURL: "://bad-url", HTTPClient: errClient.HTTPClient}, err: "failed to build request URL"},
		{name: "transport", client: Client{BaseURL: "http://127.0.0.1", HTTPClient: errClient.HTTPClient}, err: "failed to execute request"},
	} {
		t.Run(group.name, func(t *testing.T) {
			t.Parallel()

			c := group.client

			tests := []struct {
				name string
				call func() error
			}{
				{
					name: "add-stub",
					call: func() error {
						return c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})
					},
				},
				{name: "batch-delete", call: func() error { return c.BatchDelete([]uuid.UUID{uuid.New()}) }},
				{name: "upload-descriptors", call: func() error { return c.UploadDescriptors([]*descriptorpb.FileDescriptorProto{{Name: &name}}) }},
				{
					name: "fetch-history",
					call: func() error {
						_, err := c.FetchHistory()

						return err
					},
				},
				{name: "verify", call: func() error { return c.VerifyMethodCalled("svc", "M", 1) }},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()

					err := tc.call()
					require.Error(t, err)
					require.Contains(t, err.Error(), group.err)
				})
			}
		})
	}
}

func TestClientUsesRequestContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := Client{BaseURL: "http://127.0.0.1", Context: ctx}
	err := c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}
