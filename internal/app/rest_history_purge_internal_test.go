package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func purgeHistory(t *testing.T, srv *RestServer, session string) rest.HistoryPurged {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/history", nil)
	if session != "" {
		req.Header.Set(muxmiddleware.HeaderName, session)
	}

	rec := httptest.NewRecorder()
	srv.PurgeHistory(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var out rest.HistoryPurged
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))

	return out
}

func TestPurgeHistoryScopedToSession(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	store.Record(history.CallRecord{Service: "svc", Method: "Mine", Session: "a"})
	store.Record(history.CallRecord{Service: "svc", Method: "Mine", Session: "a"})
	store.Record(history.CallRecord{Service: "svc", Method: "Theirs", Session: "b"})
	store.Record(history.CallRecord{Service: "svc", Method: "Global"})

	srv, err := NewRestServer(t.Context(), stuber.NewBudgerigar(), &mockExtender{}, store, nil, nil, nil)
	require.NoError(t, err)

	out := purgeHistory(t, srv, "a")
	require.Equal(t, 2, out.DeletedCount)
	require.NotNil(t, out.Session)
	require.Equal(t, "a", *out.Session)

	left := store.All()
	require.Len(t, left, 2)

	for _, call := range left {
		require.NotEqual(t, "a", call.Session)
	}
}

func TestPurgeHistoryWithoutSessionClearsEverything(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	store.Record(history.CallRecord{Service: "svc", Method: "One", Session: "a"})
	store.Record(history.CallRecord{Service: "svc", Method: "Two"})

	srv, err := NewRestServer(t.Context(), stuber.NewBudgerigar(), &mockExtender{}, store, nil, nil, nil)
	require.NoError(t, err)

	out := purgeHistory(t, srv, "")
	require.Equal(t, 2, out.DeletedCount)
	require.Nil(t, out.Session)
	require.Empty(t, store.All())
}

func TestPurgeHistoryWithoutRecorder(t *testing.T) {
	t.Parallel()

	srv, err := NewRestServer(t.Context(), stuber.NewBudgerigar(), &mockExtender{}, nil, nil, nil, nil)
	require.NoError(t, err)

	out := purgeHistory(t, srv, "a")
	require.Equal(t, 0, out.DeletedCount)
}

func purgeStubs(t *testing.T, srv *RestServer, session string) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/stubs", nil)
	if session != "" {
		req.Header.Set(muxmiddleware.HeaderName, session)
	}

	rec := httptest.NewRecorder()
	srv.PurgeStubs(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestPurgeStubsScopedToSession(t *testing.T) {
	t.Parallel()

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(
		&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Session: "a"},
		&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Session: "b"},
		&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M"},
	)

	srv, err := NewRestServer(t.Context(), budgerigar, &mockExtender{}, history.NewMemoryStore(0), nil, nil, nil)
	require.NoError(t, err)

	purgeStubs(t, srv, "a")

	left := budgerigar.All()
	require.Len(t, left, 2, "a scoped purge must leave the other session and the global stubs")

	for _, stub := range left {
		require.NotEqual(t, "a", stub.Session)
	}
}

func TestPurgeStubsWithoutSessionClearsEverything(t *testing.T) {
	t.Parallel()

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(
		&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Session: "a"},
		&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M"},
	)

	srv, err := NewRestServer(t.Context(), budgerigar, &mockExtender{}, history.NewMemoryStore(0), nil, nil, nil)
	require.NoError(t, err)

	purgeStubs(t, srv, "")

	require.Empty(t, budgerigar.All())
}

func TestListHistoryReportsTheSession(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	store.Record(history.CallRecord{Service: "svc", Method: "Scoped", Session: "team-a"})
	store.Record(history.CallRecord{Service: "svc", Method: "Global"})

	srv, err := NewRestServer(t.Context(), stuber.NewBudgerigar(), &mockExtender{}, store, nil, nil, nil)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/history", nil)
	req.Header.Set(muxmiddleware.HeaderName, "team-a")

	srv.ListHistory(rec, req, rest.ListHistoryParams{})
	require.Equal(t, http.StatusOK, rec.Code)

	var out rest.HistoryList

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out, 2)

	byMethod := map[string]*string{}
	for _, call := range out {
		byMethod[*call.Method] = call.Session
	}

	require.NotNil(t, byMethod["Scoped"], "a session-scoped call must report its session")
	require.Equal(t, "team-a", *byMethod["Scoped"])
	require.Nil(t, byMethod["Global"], "a global call carries no session")
}
