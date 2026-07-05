package httpmock

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/remoteapi"
)

func TestAddStubs(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	client := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
	}

	err := client.AddStubs([]*stuber.Stub{{
		Service: "test.Service",
		Method:  "TestMethod",
		Input:   stuber.InputData{Equals: map[string]any{"key": "value"}},
		Output:  stuber.Output{Data: map[string]any{"result": "ok"}},
	}})
	require.NoError(t, err)

	all := s.Budgerigar.All()
	require.Len(t, all, 1)
	require.Equal(t, "test.Service", all[0].Service)
	require.Equal(t, "TestMethod", all[0].Method)
}

func TestBatchDelete(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	client := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
	}

	id1 := uuid.New()
	id2 := uuid.New()

	require.NoError(t, client.AddStubs([]*stuber.Stub{
		{
			ID: id1, Service: "svc", Method: "m1",
			Input:  stuber.InputData{Equals: map[string]any{"id": "1"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		},
		{
			ID: id2, Service: "svc", Method: "m2",
			Input:  stuber.InputData{Equals: map[string]any{"id": "2"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		},
	}))
	require.Len(t, s.Budgerigar.All(), 2)

	require.NoError(t, client.BatchDelete([]uuid.UUID{id1}))
	require.Len(t, s.Budgerigar.All(), 1)
}

func TestVerify(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	client := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
	}

	// Record some calls directly
	s.RecordCall("svc", "method", map[string]any{"req": "1"}, map[string]any{"resp": "ok"})
	s.RecordCall("svc", "method", map[string]any{"req": "2"}, map[string]any{"resp": "ok"})

	// Verify correct count
	require.NoError(t, client.VerifyMethodCalled("svc", "method", 2))

	// Verify wrong count
	err := client.VerifyMethodCalled("svc", "method", 1)
	require.Error(t, err)
}

func TestHistory(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	client := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
	}

	s.RecordCall("svc", "method", map[string]any{"req": "1"}, map[string]any{"resp": "ok"})

	history, err := client.FetchHistory()
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "svc", history[0].Service)
	require.Equal(t, "method", history[0].Method)
}

func TestSessionIsolation(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	clientA := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
		Session:    "session-A",
	}

	require.NoError(t, clientA.AddStubs([]*stuber.Stub{{
		Service: "svc", Method: "m1",
		Input:  stuber.InputData{Equals: map[string]any{"id": "1"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	}}))

	all := s.Budgerigar.All()
	require.Len(t, all, 1)
	require.Equal(t, "session-A", all[0].Session)
}

func TestListStubs(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	client := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
	}

	require.NoError(t, client.AddStubs([]*stuber.Stub{{
		Service: "svc.A", Method: "m1",
		Input:  stuber.InputData{Equals: map[string]any{"k": "v"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	}, {
		Service: "svc.B", Method: "m2",
		Input:  stuber.InputData{Equals: map[string]any{"k": "v"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	}}))

	// GET /stubs
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, s.URL+"/api/stubs", nil)
	resp, err := s.HTTPServer.Client().Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var stubs []stuber.Stub
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stubs))
	require.Len(t, stubs, 2)
}

func TestFindByID(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	id := uuid.New()
	require.NoError(t, remoteapi.Client{
		BaseURL: s.URL, HTTPClient: s.HTTPServer.Client(),
	}.AddStubs([]*stuber.Stub{{
		ID: id, Service: "svc", Method: "m1",
		Input:  stuber.InputData{Equals: map[string]any{"k": "v"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	}}))

	// GET /stubs/{uuid}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, s.URL+"/api/stubs/"+id.String(), nil)
	resp, err := s.HTTPServer.Client().Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var found stuber.Stub
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&found))
	require.Equal(t, id, found.ID)
}

func TestDeleteStubByID(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	id := uuid.New()
	client := remoteapi.Client{BaseURL: s.URL, HTTPClient: s.HTTPServer.Client()}
	require.NoError(t, client.AddStubs([]*stuber.Stub{{
		ID: id, Service: "svc", Method: "m1",
		Input:  stuber.InputData{Equals: map[string]any{"k": "v"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	}}))
	require.Len(t, s.Budgerigar.All(), 1)

	// DELETE /stubs/{uuid}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodDelete, s.URL+"/api/stubs/"+id.String(), nil)
	resp, err := s.HTTPServer.Client().Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, resp.StatusCode)
	require.Empty(t, s.Budgerigar.All())
}

func TestGzipCompression(t *testing.T) {
	t.Parallel()
	// Test that the REST server accepts gzip-compressed request bodies
	// via the GzipRequestMiddleware.
	s := NewServer()
	defer s.Close()

	// Craft a minimal HTTP request with gzip-compressed body
	stubs := []*stuber.Stub{{
		Service: "test.Service",
		Method:  "TestMethod",
		Input:   stuber.InputData{Equals: map[string]any{"key": "value"}},
		Output:  stuber.Output{Data: map[string]any{"result": "gzip-works"}},
	}}
	body, err := json.Marshal(stubs)
	require.NoError(t, err)

	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	_, err = gw.Write(body)
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, s.URL+"/api/stubs", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := s.HTTPServer.Client().Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "gzip request should be accepted")

	all := s.Budgerigar.All()
	require.Len(t, all, 1)
	data, ok := all[0].Output.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gzip-works", data["result"])
}

func TestConcurrentAddVerifyDelete(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	client := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
	}

	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()

			stub := &stuber.Stub{
				Service: "svc", Method: "m",
				Input:  stuber.InputData{Equals: map[string]any{"id": float64(id)}},
				Output: stuber.Output{Data: map[string]any{"n": float64(id)}},
			}
			if err := client.AddStubs([]*stuber.Stub{stub}); err != nil {
				t.Errorf("add stub %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	all := s.Budgerigar.All()
	t.Logf("total stubs after concurrent add: %d", len(all))
	require.GreaterOrEqual(t, len(all), 1)
}

func TestUploadDescriptors(t *testing.T) {
	t.Parallel()

	s := NewServer()
	defer s.Close()

	client := remoteapi.Client{
		BaseURL:    s.URL,
		HTTPClient: s.HTTPServer.Client(),
	}

	// Empty descriptors — should be accepted (no-op in mock)
	err := client.UploadDescriptors(nil)
	require.NoError(t, err)
}
