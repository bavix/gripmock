package sdk

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRemoteHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, "localhost:4770", normalizeRemoteAddr(" localhost:4770/ "))
	require.Equal(t, "localhost:4770", normalizeRemoteAddr("localhost:4770"))

	require.Equal(t, "http://localhost:4771", normalizeRemoteRestURL("localhost:4771"))
	require.Equal(t, "https://x.local", normalizeRemoteRestURL("https://x.local/"))
	require.Empty(t, normalizeRemoteRestURL(""))
}

func TestRemoteSetOpErrKeepsFirstError(t *testing.T) {
	t.Parallel()

	m := &remoteMock{}
	first := errors.New("first")   //nolint:err113
	second := errors.New("second") //nolint:err113

	m.setOpErr(first)
	m.setOpErr(second)

	require.ErrorIs(t, m.getOpErr(), first)
}

func TestRemoteArmSessionTTLNoSessionNoTimer(t *testing.T) {
	t.Parallel()

	m := &remoteMock{sessionTTL: time.Millisecond}
	m.armSessionTTL()
	require.Nil(t, m.ttlTimer)
}

func TestRemoteArmSessionTTLTriggersOwnedCleanup(t *testing.T) {
	t.Parallel()

	called := make(chan struct{}, 1)

	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/stubs/batchDelete" {
			called <- struct{}{}

			w.WriteHeader(http.StatusOK)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer rest.Close()

	m := &remoteMock{
		restBaseURL: rest.URL,
		httpClient:  rest.Client(),
		session:     "A",
		sessionTTL:  10 * time.Millisecond,
		stubIDs:     []uuid.UUID{uuid.New()},
	}

	m.armSessionTTL()
	t.Cleanup(func() {
		if m.ttlTimer != nil {
			m.ttlTimer.Stop()
		}
	})

	select {
	case <-called:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected TTL cleanup batch delete call")
	}
}

func TestRemoteArmSessionTTLStoresCleanupError(t *testing.T) {
	t.Parallel()

	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/stubs/batchDelete" {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer rest.Close()

	m := &remoteMock{
		restBaseURL: rest.URL,
		httpClient:  rest.Client(),
		session:     "A",
		sessionTTL:  10 * time.Millisecond,
		stubIDs:     []uuid.UUID{uuid.New()},
	}

	m.armSessionTTL()
	t.Cleanup(func() {
		if m.ttlTimer != nil {
			m.ttlTimer.Stop()
		}
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		err := m.getOpErr()
		if err != nil {
			require.Contains(t, err.Error(), "session TTL cleanup failed")

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("expected TTL cleanup error to be stored")
}
