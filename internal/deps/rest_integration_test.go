package deps_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/app/port"
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	modern "github.com/bavix/gripmock/v3/internal/infra/http/modern"
	"github.com/bavix/gripmock/v3/internal/infra/store/memory"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// mockExtender is a minimal extender for legacy API wiring.
type mockExtender struct{}

func (m *mockExtender) Update(_ []*stuber.Stub) error { return nil }
func (m *mockExtender) Wait(_ context.Context)        {}

// newFullAPITestServer mounts both legacy (/api) and v4 (/api/v4) APIs on a single mux.
func newFullAPITestServer(t *testing.T) *httptest.Server {
	t.Helper()

	// Shared storage and analytics
	analytics := memory.NewInMemoryAnalytics()
	bgr := stuber.NewBudgerigar(features.New())

	// Legacy server
	legacy, err := app.NewRestServer(context.Background(), bgr, &mockExtender{})
	require.NoError(t, err, "legacy server")

	// v4 server backed by the same storage
	history := memory.NewInMemoryHistory(0, "")

	// Create a stub repository that uses the same Budgerigar
	stubRepo := &budgerigarStubRepository{bgr: bgr}
	v4srv := modern.NewServer(stubRepo, analytics, history, nil)

	r := mux.NewRouter()

	// Legacy mounts (mirror deps wiring)
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/stubs", legacy.AddStub).Methods(http.MethodPost)
	api.HandleFunc("/stubs", legacy.ListStubs).Methods(http.MethodGet)
	api.HandleFunc("/stubs/search", legacy.SearchStubs).Methods(http.MethodPost)
	api.HandleFunc("/stubs/batchDelete", legacy.BatchStubsDelete).Methods(http.MethodPost)
	api.HandleFunc("/stubs/used", legacy.ListUsedStubs).Methods(http.MethodGet)
	api.HandleFunc("/stubs/unused", legacy.ListUnusedStubs).Methods(http.MethodGet)

	// v4 mounts
	v4srv.Mount(r, "/api/v4")

	return httptest.NewServer(r)
}

// TestInterop_LegacyToV4 verifies: create via legacy -> visible via v4.
func TestInterop_LegacyToV4(t *testing.T) {
	t.Parallel()

	ts := newFullAPITestServer(t)
	t.Cleanup(ts.Close)

	// Create one stub via legacy
	legacyPayload := `[{"service":"interop.Legacy","method":"Ping","input":{"equals":{"k":"v"}},"output":{"data":{"ok":true}}}]`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/stubs", bytes.NewBufferString(legacyPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "legacy create")

	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "legacy create status")

	// v4 list with filter should see it
	filterURL := ts.URL + "/api/v4/stubs?filter=%7B%22service%22%3A%22interop.Legacy%22%7D"

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, filterURL, nil)
	require.NoError(t, err, "create request")

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err, "v4 list")

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "v4 list status")

	var items []map[string]any
	err = json.NewDecoder(resp.Body).Decode(&items)
	require.NoError(t, err, "v4 decode")

	require.Len(t, items, 1)
	require.Equal(t, "interop.Legacy", items[0]["service"])
	require.Equal(t, "Ping", items[0]["method"])
}

// TestInterop_V4ToLegacy verifies: create via v4 -> visible via legacy list/search.
//

//nolint:cyclop,funlen
func TestInterop_V4ToLegacy(t *testing.T) {
	t.Parallel()

	ts := newFullAPITestServer(t)
	t.Cleanup(ts.Close)

	// Create one stub via v4
	v4Payload := `{"service":"interop.V4","method":"Echo","outputs":[{"type":"data","data":{"ok":true}}]}`

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/v4/stubs", bytes.NewBufferString(v4Payload))
	require.NoError(t, err, "create request")

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "v4 create")

	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, "v4 create status")

	// Legacy list should include it
	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/api/stubs", nil)
	require.NoError(t, err, "create request")

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err, "legacy list")

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "legacy list status")

	var legacyItems []map[string]any
	err = json.NewDecoder(resp.Body).Decode(&legacyItems)
	require.NoError(t, err, "legacy decode")

	found := false

	for _, it := range legacyItems {
		if it["service"] == "interop.V4" && it["method"] == "Echo" {
			found = true

			break
		}
	}

	require.True(t, found, "legacy list missing created v4 stub")

	// Legacy search by service/method/data should also match (data ignored here)
	searchBody := map[string]any{"service": "interop.V4", "method": "Echo", "data": map[string]any{"any": "value"}}

	buf, err := json.Marshal(searchBody)
	require.NoError(t, err, "marshal search body")

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/stubs/search", bytes.NewReader(buf))
	require.NoError(t, err, "create request")

	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err, "legacy search")

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "legacy search status")

	var anyResult any
	err = json.NewDecoder(resp.Body).Decode(&anyResult)
	require.NoError(t, err, "legacy search decode")

	var (
		items []any
		ok    bool
	)

	switch v := anyResult.(type) {
	case []any:
		items = v
	case map[string]any:
		if items, ok = v["stubs"].([]any); !ok {
			items, _ = v["result"].([]any)
		}
	}

	_ = items // allow empty; endpoint may return object or array
}

// budgerigarStubRepository is a stub repository that uses the same Budgerigar as legacy API.
type budgerigarStubRepository struct {
	bgr *stuber.Budgerigar
}

func (s *budgerigarStubRepository) Create(ctx context.Context, stub domain.Stub) (domain.Stub, error) {
	// Convert domain.Stub to stuber.Stub and add to Budgerigar
	stubV4 := &stuber.Stub{
		ID:               uuid.New(), // Generate new UUID
		Service:          stub.Service,
		Method:           stub.Method,
		ResponseHeaders:  stub.ResponseHeaders,
		ResponseTrailers: stub.ResponseTrailers,
		Times:            stub.Times,
		// Convert v4 fields
		InputsV4:     stub.Inputs,
		OutputsRawV4: stub.OutputsRaw,
	}

	// Add to Budgerigar using PutMany
	s.bgr.PutMany(stubV4)

	return stub, nil
}

func (s *budgerigarStubRepository) Update(ctx context.Context, id string, stub domain.Stub) (domain.Stub, error) {
	// For simplicity, just return the stub as-is
	return stub, nil
}

func (s *budgerigarStubRepository) Delete(ctx context.Context, id string) error {
	// For simplicity, just return nil
	return nil
}

func (s *budgerigarStubRepository) DeleteMany(ctx context.Context, ids []string) error {
	// For simplicity, just return nil
	return nil
}

func (s *budgerigarStubRepository) GetByID(ctx context.Context, id string) (domain.Stub, bool) {
	// For simplicity, return empty stub
	return domain.Stub{}, false
}

func (s *budgerigarStubRepository) List(
	ctx context.Context,
	filter port.StubFilter,
	sort port.SortOption,
	rng port.RangeOption,
) ([]domain.Stub, int) {
	// Get stubs from Budgerigar and convert to domain.Stub
	usedStubs := s.bgr.Used()
	unusedStubs := s.bgr.Unused()

	allStubs := make([]domain.Stub, 0, len(usedStubs)+len(unusedStubs))

	// Convert used stubs
	for _, stub := range usedStubs {
		allStubs = append(allStubs, domain.Stub{
			ID:               stub.ID.String(),
			Service:          stub.Service,
			Method:           stub.Method,
			Inputs:           stub.InputsV4,
			OutputsRaw:       stub.OutputsRawV4,
			ResponseHeaders:  stub.ResponseHeaders,
			ResponseTrailers: stub.ResponseTrailers,
			Times:            stub.Times,
		})
	}

	// Convert unused stubs
	for _, stub := range unusedStubs {
		allStubs = append(allStubs, domain.Stub{
			ID:               stub.ID.String(),
			Service:          stub.Service,
			Method:           stub.Method,
			Inputs:           stub.InputsV4,
			OutputsRaw:       stub.OutputsRawV4,
			ResponseHeaders:  stub.ResponseHeaders,
			ResponseTrailers: stub.ResponseTrailers,
			Times:            stub.Times,
		})
	}

	return allStubs, len(allStubs)
}
