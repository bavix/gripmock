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

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type errRoundTripper struct{}

func (errRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport failed")
}

func newTestServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(handler))
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
		require.Equal(t, "/api/stubs", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))

		var payload []map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Len(t, payload, 1)
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/stubs/batchDelete", r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
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

func TestClientUploadDescriptors(t *testing.T) {
	t.Parallel()

	// Arrange
	name := "svc.proto"
	var called bool
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/descriptors", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))

		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var fds descriptorpb.FileDescriptorSet
		require.NoError(t, proto.Unmarshal(raw, &fds))
		require.Len(t, fds.File, 1)
		require.Equal(t, name, fds.File[0].GetName())

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
		require.Equal(t, "/api/history", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[{\"service\":\"svc\",\"method\":\"M\",\"request\":{\"x\":1},\"response\":{\"ok\":true},\"stubId\":\"550e8400-e29b-41d4-a716-446655440000\",\"timestamp\":\"" + now + "\"}]"))
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
	require.Equal(t, float64(1), history[0].Request["x"])
}

func TestClientVerifyMethodCalledBadRequest(t *testing.T) {
	t.Parallel()

	// Arrange
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/verify", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad count"}`))
	})

	c := newTestClient(ts, "")

	// Act
	err := c.VerifyMethodCalled("svc", "M", 2)

	// Assert
	require.Error(t, err)
	badReq, ok := err.(VerifyBadRequestError)
	require.True(t, ok)
	require.Equal(t, "bad count", badReq.Error())
}

func TestClientAddStubUsesDefaultHTTPClientWhenNil(t *testing.T) {
	t.Parallel()

	// Arrange
	var called bool
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/stubs", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
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
		require.Equal(t, "/api/stubs", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		var payload []map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
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
	require.Equal(t, "", ptrOrZero[string](nil))
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
		require.Equal(t, "/api/stubs", r.URL.Path)
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
	err := c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"bad": make(chan int)}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to marshal stubs")
}

func TestClientBatchDeleteErrorStatus(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/stubs/batchDelete", r.URL.Path)
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
			require.Equal(t, "/api/descriptors", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		})

		c := newTestClient(ts, "")
		err := c.UploadDescriptors([]*descriptorpb.FileDescriptorProto{{Name: &name}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "upload descriptors failed with status 500")
	})
}

func TestClientFetchHistoryErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("server-status", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/api/history", r.URL.Path)
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
			require.Equal(t, "/api/history", r.URL.Path)
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
			require.Equal(t, "/api/history", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{}}))
		})

		c := newTestClient(ts, "")
		history, err := c.FetchHistory()
		require.NoError(t, err)
		require.Len(t, history, 1)
		require.Equal(t, "", history[0].Service)
		require.Equal(t, "", history[0].Method)
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
			require.Equal(t, "/api/verify", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		})

		c := newTestClient(ts, "")
		require.NoError(t, c.VerifyMethodCalled("svc", "M", 1))
	})

	t.Run("bad-request-invalid-json", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/api/verify", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("not-json"))
		})

		c := newTestClient(ts, "")
		err := c.VerifyMethodCalled("svc", "M", 1)
		var badReq VerifyBadRequestError
		require.True(t, errors.As(err, &badReq))
		require.Equal(t, "verification failed", badReq.Error())
	})

	t.Run("server-status", func(t *testing.T) {
		t.Parallel()

		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/api/verify", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		})

		c := newTestClient(ts, "")
		err := c.VerifyMethodCalled("svc", "M", 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "verify request failed with status 500")
	})
}

func TestClientURLBuildErrors(t *testing.T) {
	t.Parallel()

	c := Client{BaseURL: "://bad-url", HTTPClient: &http.Client{Transport: errRoundTripper{}}}
	name := "svc.proto"

	tests := []struct {
		name       string
		call       func() error
		errContain string
	}{
		{
			name: "add-stub",
			call: func() error {
				return c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})
			},
			errContain: "failed to build request URL",
		},
		{
			name:       "batch-delete",
			call:       func() error { return c.BatchDelete([]uuid.UUID{uuid.New()}) },
			errContain: "failed to build request URL",
		},
		{
			name:       "upload-descriptors",
			call:       func() error { return c.UploadDescriptors([]*descriptorpb.FileDescriptorProto{{Name: &name}}) },
			errContain: "failed to build request URL",
		},
		{
			name: "fetch-history",
			call: func() error {
				_, err := c.FetchHistory()
				return err
			},
			errContain: "failed to build request URL",
		},
		{
			name:       "verify",
			call:       func() error { return c.VerifyMethodCalled("svc", "M", 1) },
			errContain: "failed to build request URL",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.call()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContain)
		})
	}
}

func TestClientTransportErrors(t *testing.T) {
	t.Parallel()

	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: errRoundTripper{}}}
	name := "svc.proto"

	tests := []struct {
		name       string
		call       func() error
		errContain string
	}{
		{
			name: "add-stub",
			call: func() error {
				return c.AddStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})
			},
			errContain: "failed to execute request",
		},
		{
			name:       "batch-delete",
			call:       func() error { return c.BatchDelete([]uuid.UUID{uuid.New()}) },
			errContain: "failed to execute request",
		},
		{
			name:       "upload-descriptors",
			call:       func() error { return c.UploadDescriptors([]*descriptorpb.FileDescriptorProto{{Name: &name}}) },
			errContain: "failed to execute request",
		},
		{
			name: "fetch-history",
			call: func() error {
				_, err := c.FetchHistory()
				return err
			},
			errContain: "failed to execute request",
		},
		{
			name:       "verify",
			call:       func() error { return c.VerifyMethodCalled("svc", "M", 1) },
			errContain: "failed to execute request",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.call()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContain)
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
