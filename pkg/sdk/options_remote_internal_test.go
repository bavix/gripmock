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

func TestDeriveAndExtractHostHelpers(t *testing.T) {
	t.Parallel()

	// deriveRestURLFromGrpcAddr branches
	require.Equal(t, "http://localhost:4771", deriveRestURLFromGrpcAddr("localhost:4770"))
	require.Equal(t, "http://127.0.0.1:4771", deriveRestURLFromGrpcAddr("127.0.0.1:8888"))
	require.Equal(t, "http://[bad::addr]:4771", deriveRestURLFromGrpcAddr("bad::addr"))

	// extractHost branches
	require.Equal(t, "example.com", extractHost("example.com:9090"))
	require.Equal(t, "http://api.local:8080", extractHost("http://api.local:8080"))
	require.Equal(t, "", extractHost(""))
}

func TestNormalizeRemoteHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, "localhost:4770", normalizeRemoteAddr(" localhost:4770/ "))
	require.Equal(t, "localhost:4770", normalizeRemoteAddr("localhost:4770"))

	require.Equal(t, "http://localhost:4771", normalizeRemoteRestURL("localhost:4771"))
	require.Equal(t, "https://x.local", normalizeRemoteRestURL("https://x.local/"))
	require.Equal(t, "", normalizeRemoteRestURL(""))
}

func TestRemoteMethodKeyHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, "svc/M", methodKey("svc", "M"))

	svc, m, ok := splitMethodKey("svc/M")
	require.True(t, ok)
	require.Equal(t, "svc", svc)
	require.Equal(t, "M", m)

	_, _, ok = splitMethodKey("svc")
	require.False(t, ok)
	_, _, ok = splitMethodKey("/M")
	require.False(t, ok)
	_, _, ok = splitMethodKey("svc/")
	require.False(t, ok)
}

func TestRemoteSetOpErrKeepsFirstError(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{}
	first := errors.New("first")
	second := errors.New("second")

	// Act
	m.setOpErr(first)
	m.setOpErr(second)

	// Assert
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

	// Arrange
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

	// Act
	m.armSessionTTL()
	t.Cleanup(func() {
		if m.ttlTimer != nil {
			m.ttlTimer.Stop()
		}
	})

	// Assert
	select {
	case <-called:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected TTL cleanup batch delete call")
	}
}

func TestRemoteArmSessionTTLStoresCleanupError(t *testing.T) {
	t.Parallel()

	// Arrange
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

	// Act
	m.armSessionTTL()
	t.Cleanup(func() {
		if m.ttlTimer != nil {
			m.ttlTimer.Stop()
		}
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := m.getOpErr(); err != nil {
			require.Contains(t, err.Error(), "session TTL cleanup failed")
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("expected TTL cleanup error to be stored")
}
