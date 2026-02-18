package sdk

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type captureTestingT struct {
	ctx      context.Context
	errors   []string
	failed   int
	cleanups int
}

func (c *captureTestingT) Error(args ...any) {
	c.errors = append(c.errors, stringify(args...))
}

func (c *captureTestingT) Fail() {
	c.failed++
}

func (c *captureTestingT) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}

	return c.ctx
}

func (c *captureTestingT) Cleanup(_ func()) {
	c.cleanups++
}

func stringify(args ...any) string {
	b, _ := json.Marshal(args)

	return string(b)
}

func TestRemoteMock_CleanupStubs_BySession(t *testing.T) {
	t.Parallel()

	// Arrange
	idA1 := uuid.New()
	idA2 := uuid.New()
	idB := uuid.New()
	var deleted []uuid.UUID

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/stubs":
			require.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": idA1.String(), "session": "A"},
				{"id": idB.String(), "session": "B"},
				{"id": idA2.String(), "session": "A"},
			})
		case "/api/stubs/batchDelete":
			require.Equal(t, http.MethodPost, r.Method)
			var got []uuid.UUID
			require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
			deleted = got
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	m := &remoteMock{
		restBaseURL: srv.URL,
		httpClient:  srv.Client(),
		session:     "A",
	}

	// Act
	require.NoError(t, m.cleanupStubs())

	// Assert
	require.ElementsMatch(t, []uuid.UUID{idA1, idA2}, deleted)
}

func TestRemoteMock_CleanupStubs_NoSessionUsesOwnedIDs(t *testing.T) {
	t.Parallel()

	// Arrange
	id1 := uuid.New()
	id2 := uuid.New()
	var deleted []uuid.UUID

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/stubs/batchDelete", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		var got []uuid.UUID
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		deleted = got
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := &remoteMock{
		restBaseURL: srv.URL,
		httpClient:  srv.Client(),
		stubIDs:     []uuid.UUID{id1, id2},
	}

	// Act
	require.NoError(t, m.cleanupStubs())

	// Assert
	require.ElementsMatch(t, []uuid.UUID{id1, id2}, deleted)
}

func TestRemoteMock_ArmSessionTTL_TriggersCleanup(t *testing.T) {
	t.Parallel()

	// Arrange
	var listCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/stubs", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		listCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	m := &remoteMock{
		restBaseURL: srv.URL,
		httpClient:  srv.Client(),
		session:     "A",
		sessionTTL:  10 * time.Millisecond,
	}

	// Act
	m.armSessionTTL()
	t.Cleanup(func() {
		if m.ttlTimer != nil {
			m.ttlTimer.Stop()
		}
	})

	// Assert
	require.Eventually(t, func() bool {
		return listCalls.Load() >= 1
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestRunRemote_CleansSessionOnStart(t *testing.T) {
	t.Parallel()

	// Arrange
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	hs := health.NewServer()
	hs.SetServingStatus("gripmock", healthgrpc.HealthCheckResponse_SERVING)

	gs := grpc.NewServer()
	healthgrpc.RegisterHealthServer(gs, hs)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)
	t.Cleanup(func() { _ = lis.Close() })

	idA := uuid.New()
	var listCalls atomic.Int32
	var deleteCalls atomic.Int32

	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/stubs":
			listCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": idA.String(), "session": "A"}})
		case "/api/stubs/batchDelete":
			deleteCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rest.Close()

	// Act
	m, err := runRemote(t.Context(), &options{
		remoteAddr:     lis.Addr().String(),
		remoteRestURL:  rest.URL,
		httpClient:     rest.Client(),
		session:        "A",
		healthyTimeout: time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close() })

	// Assert
	require.GreaterOrEqual(t, listCalls.Load(), int32(1))
	require.GreaterOrEqual(t, deleteCalls.Load(), int32(1))
}

func TestRemoteMock_CloseCleanupError(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{restBaseURL: "://bad-url", session: "A", httpClient: http.DefaultClient}

	// Act
	err := m.Close()

	// Assert
	require.Error(t, err)
}

func TestRemoteHistoryAndVerifier(t *testing.T) {
	t.Parallel()

	// Arrange
	ts := &captureTestingT{}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/history":
			require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[{\"service\":\"svc\",\"method\":\"M\",\"request\":{\"x\":1},\"response\":{\"y\":2},\"error\":\"\",\"stubId\":\"id\",\"timestamp\":\"" + now + "\"}]"))
		case "/api/verify":
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			if req["expectedCount"] == float64(3) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"bad count"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rest.Close()

	m := &remoteMock{restBaseURL: rest.URL, httpClient: rest.Client(), session: "A"}

	// Act
	h := m.History()
	all := h.All()
	count := h.Count()
	filtered := h.FilterByMethod("svc", "M")

	v := m.Verify()
	v.Total(ts, 2)
	v.Method("svc", "M").Called(ts, 3)
	v.Method("svc", "M").Never(ts)
	v.VerifyStubTimes(ts)

	m.expectedTotal.Store(2)
	errTimes := v.VerifyStubTimesErr()

	// Assert
	require.Len(t, all, 1)
	require.Equal(t, 1, count)
	require.Len(t, filtered, 1)
	require.Error(t, errTimes)
	require.GreaterOrEqual(t, ts.failed, 2)
}

func TestRemoteAddStubAndCleanupOwnedIDs(t *testing.T) {
	t.Parallel()

	// Arrange
	var added int32
	var listed int32
	var deleted int32
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/stubs":
			if r.Method == http.MethodPost {
				require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
				added++
				w.WriteHeader(http.StatusOK)
				return
			}

			require.Equal(t, http.MethodGet, r.Method)
			listed++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"11111111-1111-1111-1111-111111111111","session":"A"}]`))
		case "/api/stubs/batchDelete":
			require.Equal(t, http.MethodPost, r.Method)
			deleted++
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rest.Close()

	m := &remoteMock{restBaseURL: rest.URL, httpClient: rest.Client(), session: "A"}
	s := m.Stub("svc", "M")

	// Act
	s.Reply(stuber.Output{Data: map[string]any{"ok": true}}).Times(2).Commit()
	err := m.Close()

	// Assert
	require.NoError(t, err)
	require.Equal(t, int32(1), added)
	require.Equal(t, int32(1), listed)
	require.Equal(t, int32(1), deleted)
	require.Equal(t, int32(2), m.expectedTotal.Load())
}

func TestRemoteMock_BatchDeleteAndSessionDeleteErrors(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{restBaseURL: "://bad-url", httpClient: http.DefaultClient, session: "A"}

	// Act
	errSession := m.deleteSessionStubs()
	errBatch := m.batchDelete([]uuid.UUID{uuid.New()})

	// Assert
	require.Error(t, errSession)
	require.Error(t, errBatch)
}

func TestSessionInterceptors(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := t.Context()
	unaryCalled := false
	streamCalled := false

	// Act
	errUnary := sessionUnaryInterceptor("A")(ctx, "/svc/M", nil, nil, nil, func(invCtx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		unaryCalled = true
		md, ok := metadata.FromOutgoingContext(invCtx)
		require.True(t, ok)
		require.Equal(t, "A", md.Get("x-gripmock-session")[0])
		return nil
	})

	_, errStream := sessionStreamInterceptor("A")(ctx, &grpc.StreamDesc{}, nil, "/svc/M", func(streamCtx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		streamCalled = true
		md, ok := metadata.FromOutgoingContext(streamCtx)
		require.True(t, ok)
		require.Equal(t, "A", md.Get("x-gripmock-session")[0])
		return nil, nil
	})

	// Assert
	require.NoError(t, errUnary)
	require.NoError(t, errStream)
	require.True(t, unaryCalled)
	require.True(t, streamCalled)
}

func TestTimeoutInterceptors(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := t.Context()
	timeout := 50 * time.Millisecond
	unaryCalled := false
	streamCalled := false

	// Act
	errUnary := timeoutUnaryInterceptor(timeout)(ctx, "/svc/M", nil, nil, nil, func(invCtx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		unaryCalled = true
		deadline, ok := invCtx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(timeout), deadline, 30*time.Millisecond)

		return nil
	})

	_, errStream := timeoutStreamInterceptor(timeout)(ctx, &grpc.StreamDesc{}, nil, "/svc/M", func(streamCtx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		streamCalled = true
		deadline, ok := streamCtx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(timeout), deadline, 30*time.Millisecond)

		return nil, nil
	})

	// Assert
	require.NoError(t, errUnary)
	require.NoError(t, errStream)
	require.True(t, unaryCalled)
	require.True(t, streamCalled)
}

func TestRemoteHelpers(t *testing.T) {
	t.Parallel()

	// Arrange
	now := time.Now().UTC()

	// Act
	_ = ptrVal(nil)
	_ = ptrMapVal(nil)
	_ = ptrTimeVal(nil)
	val := ptrVal(ptr("x"))
	mapVal := ptrMapVal(&map[string]any{"k": "v"})
	timeVal := ptrTimeVal(&now)

	// Assert
	require.Equal(t, "x", val)
	require.Equal(t, "v", mapVal["k"])
	require.Equal(t, now, timeVal)
}

func ptr[T any](v T) *T {
	return &v
}

func TestRemoteMethodVerifierRequestCreationError(t *testing.T) {
	t.Parallel()

	// Arrange
	ts := &captureTestingT{}
	mv := &remoteMethodVerifier{mock: &remoteMock{restBaseURL: "://bad-url", httpClient: http.DefaultClient}, service: "svc", method: "M"}

	// Act
	mv.Called(ts, 1)

	// Assert
	require.Equal(t, 1, ts.failed)
	require.NotEmpty(t, ts.errors)
}

func TestRemoteHistoryDecodeErrorAndStatus(t *testing.T) {
	t.Parallel()

	// Arrange
	statusSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer statusSrv.Close()

	decodeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer decodeSrv.Close()

	// Act
	allStatus := (&remoteMock{restBaseURL: statusSrv.URL, httpClient: statusSrv.Client()}).History().All()
	allDecode := (&remoteMock{restBaseURL: decodeSrv.URL, httpClient: decodeSrv.Client()}).History().All()

	// Assert
	require.Nil(t, allStatus)
	require.Nil(t, allDecode)
}

func TestRemoteAddStubPanicsOnBadURL(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{restBaseURL: "://bad-url", httpClient: http.DefaultClient}

	// Act
	fn := func() {
		m.addStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Code: ptr(codes.OK)}})
	}

	// Assert
	require.Panics(t, fn)
}
