package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/testkit"
)

type captureTestingT struct {
	ctx context.Context

	failed   atomic.Int32
	cleanups atomic.Int32

	mu      sync.Mutex
	errors  []string
	cleanup []func()
}

func (c *captureTestingT) Error(args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = append(c.errors, fmt.Sprint(args...))
}

func (c *captureTestingT) Fail() {
	c.failed.Add(1)
}

func (c *captureTestingT) Context() context.Context {
	return c.ctx
}

func (c *captureTestingT) Cleanup(_ func()) {
	c.cleanups.Add(1)
}

func (c *captureTestingT) CleanupWithRun(fn func()) {
	c.cleanups.Add(1)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanup = append(c.cleanup, fn)
}

func (c *captureTestingT) RunCleanups() {
	c.mu.Lock()
	cleanups := slices.Clone(c.cleanup)
	c.mu.Unlock()

	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}

func (c *captureTestingT) Failed() int {
	return int(c.failed.Load())
}

func (c *captureTestingT) Errors() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Clone(c.errors)
}

func (c *captureTestingT) Cleanups() int {
	return int(c.cleanups.Load())
}

func (c *captureTestingT) FirstError() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.errors) == 0 {
		return ""
	}

	return c.errors[0]
}

type manualCleanupT struct {
	captureTestingT
}

func newRemoteServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	return server
}

func newRemoteMockForServer(server *httptest.Server, session string) *remoteMock {
	return &remoteMock{restBaseURL: server.URL, httpClient: server.Client(), session: session}
}

func newManualCleanupT(ctx context.Context) *manualCleanupT {
	return &manualCleanupT{captureTestingT: captureTestingT{ctx: ctx}}
}

func (m *manualCleanupT) Cleanup(fn func()) {
	m.CleanupWithRun(fn)
}

func TestRemoteHistoryAndVerifier(t *testing.T) {
	t.Parallel()

	// Arrange
	ts := &captureTestingT{ctx: t.Context()}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/history":
			require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[{\"service\":\"svc\",\"method\":\"M\",\"request\":{\"x\":1},\"response\":{\"y\":2},\"error\":\"\",\"stubId\":\"550e8400-e29b-41d4-a716-446655440000\",\"timestamp\":\"" + now + "\"}]"))
		case "/api/verify":
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			if req["expectedCount"] == float64(2) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"bad count"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	m := newRemoteMockForServer(rest, "A")

	// Act
	h := m.History()
	all := h.All()
	count := h.Count()
	filtered := h.FilterByMethod("svc", "M")

	v := m.Verify()
	v.Total(ts, 2)
	v.Method(By("/svc/M")).Called(ts, 3)
	v.Method(By("/svc/M")).Never(ts)
	v.VerifyStubTimes(ts)

	m.expectedTotal.Store(2)
	m.expectedMu.Lock()
	m.expectedByMth = map[string]int{methodKey("svc", "M"): 2}
	m.expectedMu.Unlock()
	errTimes := v.VerifyStubTimesErr()

	// Assert
	require.Len(t, all, 1)
	require.Equal(t, 1, count)
	require.Len(t, filtered, 1)
	require.Error(t, errTimes)
	require.GreaterOrEqual(t, ts.Failed(), 1)
}

func TestRemoteConnAndAddrAccessors(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{addr: "127.0.0.1:4770", conn: nil}

	// Act + Assert
	require.Equal(t, "127.0.0.1:4770", m.Addr())
	require.Nil(t, m.Conn())
}

func TestRemoteAddStubAndCleanupOwnedIDs(t *testing.T) {
	t.Parallel()

	// Arrange
	var added atomic.Int32
	var listed atomic.Int32
	var deleted atomic.Int32
	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/stubs":
			if r.Method == http.MethodPost {
				require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
				added.Add(1)
				w.WriteHeader(http.StatusOK)
				return
			}

			require.Equal(t, http.MethodGet, r.Method)
			listed.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"11111111-1111-1111-1111-111111111111","session":"A"}]`))
		case "/api/stubs/batchDelete":
			require.Equal(t, http.MethodPost, r.Method)
			deleted.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	m := newRemoteMockForServer(rest, "A")
	s := m.Stub(By("/svc/M"))

	// Act
	s.Reply(stuber.Output{Data: map[string]any{"ok": true}}).Times(2).Commit()
	err := m.Close()

	// Assert
	require.NoError(t, err)
	require.Equal(t, int32(1), added.Load())
	require.Equal(t, int32(0), listed.Load())
	require.Equal(t, int32(1), deleted.Load())
	require.Equal(t, int32(2), m.expectedTotal.Load())
}

func TestRemoteMockBatchDeleteErrors(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{restBaseURL: "://bad-url", httpClient: http.DefaultClient, session: "A"}

	// Act
	errBatch := m.batchDelete([]uuid.UUID{uuid.New()})

	// Assert
	require.Error(t, errBatch)
}

func TestRemoteAddStubNoPanicOnBadURL(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{restBaseURL: "://bad-url", httpClient: http.DefaultClient}

	// Act
	code := codes.OK
	m.addStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Code: &code}})
	err := m.getOpErr()

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to build request URL")
}

func TestRemoteAddStubHTTPFailureStoredAsError(t *testing.T) {
	t.Parallel()

	// Arrange
	var calls atomic.Int32
	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/stubs" {
			calls.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	m := newRemoteMockForServer(rest, "")

	// Act
	m.addStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})
	err := m.getOpErr()

	// Assert
	require.Equal(t, int32(1), calls.Load())
	require.Error(t, err)
	require.Contains(t, err.Error(), "add stubs failed with status 500")
	require.Equal(t, int32(0), m.expectedTotal.Load())
}

func TestRemoteAPIUsesDefaultHTTPClientWhenNil(t *testing.T) {
	t.Parallel()

	// Arrange
	var calls atomic.Int32
	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/stubs" {
			calls.Add(1)
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	m := &remoteMock{restBaseURL: rest.URL}

	// Act
	m.addStub(&stuber.Stub{ID: uuid.New(), Service: "svc", Method: "M", Output: stuber.Output{Data: map[string]any{"ok": true}}})
	err := m.getOpErr()

	// Assert
	require.Equal(t, int32(1), calls.Load())
	require.NoError(t, err)
}

func TestStubBatchRemoteSingleAddRequest(t *testing.T) {
	t.Parallel()

	grpcAddr := testkit.StartHealthGRPC(t)

	var stubPosts atomic.Int32
	var batchDeletes atomic.Int32

	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/stubs":
			require.Equal(t, http.MethodPost, r.Method)
			stubPosts.Add(1)

			var payload []map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Len(t, payload, 2)
			w.WriteHeader(http.StatusOK)
		case "/api/stubs/batchDelete":
			batchDeletes.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	mock, err := Run(t, WithRemote(grpcAddr, rest.URL), WithHealthCheckTimeout(time.Second))
	require.NoError(t, err)

	batch := NewBatch(mock)
	batch.Stub(By("/svc/M1")).Reply(Data("ok", true)).Commit()
	batch.Stub(By("/svc/M2")).Reply(Data("ok", true)).Commit()

	require.NoError(t, batch.Commit())
	require.NoError(t, mock.Close())

	require.Equal(t, int32(1), stubPosts.Load())
	require.GreaterOrEqual(t, batchDeletes.Load(), int32(1))
}

func TestRemoteMockCloseCleanupError(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{restBaseURL: "://bad-url", session: "A", httpClient: http.DefaultClient, stubIDs: []uuid.UUID{uuid.New()}}

	// Act
	err := m.Close()

	// Assert
	require.Error(t, err)
}

func TestRemoteMockCloseAlwaysClosesConnOnCleanupError(t *testing.T) {
	t.Parallel()

	// Arrange
	conn, err := grpc.NewClient("passthrough:///127.0.0.1:65535", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	m := &remoteMock{
		conn:        conn,
		restBaseURL: "://bad-url",
		session:     "A",
		httpClient:  http.DefaultClient,
		stubIDs:     []uuid.UUID{uuid.New()},
	}

	// Act
	err = m.Close()

	// Assert
	require.Error(t, err)
	require.Nil(t, m.conn)
}

func TestRemoteVerifyStubTimesErrScenarios(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		setup           func(m *remoteMock)
		handler         func(t *testing.T, verifyCalls *atomic.Int32) func(http.ResponseWriter, *http.Request)
		wantErrContains string
		wantErrIs       error
		wantVerifyCalls int32
	}{
		{
			name: "no-expected-total",
			setup: func(m *remoteMock) {
				m.expectedTotal.Store(0)
			},
			wantVerifyCalls: 0,
		},
		{
			name: "operation-error",
			setup: func(m *remoteMock) {
				m.expectedTotal.Store(1)
				m.setOpErr(context.DeadlineExceeded)
			},
			wantErrIs:       context.DeadlineExceeded,
			wantVerifyCalls: 0,
		},
		{
			name: "invalid-method-key",
			setup: func(m *remoteMock) {
				m.expectedTotal.Store(1)
				m.expectedByMth = map[string]int{"svc": 1}
			},
			wantErrContains: "invalid expected method key",
			wantVerifyCalls: 0,
		},
		{
			name: "verify-bad-request",
			setup: func(m *remoteMock) {
				m.expectedTotal.Store(2)
				m.expectedByMth = map[string]int{methodKey("svc", "M"): 2}
			},
			handler: func(t *testing.T, verifyCalls *atomic.Int32) func(http.ResponseWriter, *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "/api/verify", r.URL.Path)
					verifyCalls.Add(1)
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"message":"bad count"}`))
				}
			},
			wantErrContains: "expected 2 calls for svc/M",
			wantVerifyCalls: 1,
		},
		{
			name: "verify-multiple-methods",
			setup: func(m *remoteMock) {
				m.expectedTotal.Store(3)
				m.expectedByMth = map[string]int{
					methodKey("svc", "M"): 2,
					methodKey("svc", "N"): 1,
				}
			},
			handler: func(t *testing.T, verifyCalls *atomic.Int32) func(http.ResponseWriter, *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "/api/verify", r.URL.Path)
					verifyCalls.Add(1)

					var req struct {
						Service       string `json:"service"`
						Method        string `json:"method"`
						ExpectedCount int    `json:"expectedCount"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
					require.Equal(t, "svc", req.Service)
					if req.Method == "M" {
						require.Equal(t, 2, req.ExpectedCount)
					} else {
						require.Equal(t, "N", req.Method)
						require.Equal(t, 1, req.ExpectedCount)
					}

					w.WriteHeader(http.StatusOK)
				}
			},
			wantVerifyCalls: 2,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &remoteMock{}
			if tc.handler != nil {
				var verifyCalls atomic.Int32
				rest := newRemoteServer(t, tc.handler(t, &verifyCalls))
				m = newRemoteMockForServer(rest, "")
				if tc.setup != nil {
					tc.setup(m)
				}

				err := m.Verify().VerifyStubTimesErr()
				if tc.wantErrContains != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.wantErrContains)
				} else if tc.wantErrIs != nil {
					require.Error(t, err)
					require.ErrorIs(t, err, tc.wantErrIs)
				} else {
					require.NoError(t, err)
				}

				require.Equal(t, tc.wantVerifyCalls, verifyCalls.Load())
				return
			}

			if tc.setup != nil {
				tc.setup(m)
			}

			err := m.Verify().VerifyStubTimesErr()
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
			} else if tc.wantErrIs != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.wantErrIs)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRemoteMockCleanupOwnedIDs(t *testing.T) {
	t.Parallel()

	// Arrange
	id1 := uuid.New()
	id2 := uuid.New()
	var deleted []uuid.UUID

	srv := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/stubs/batchDelete", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		var got []uuid.UUID
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		deleted = got
		w.WriteHeader(http.StatusOK)
	})

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

func TestRunRemoteViaSDKCleanupScenarios(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		session            string
		times              int
		historyCalls       int
		verifyStatus       int
		verifyBody         string
		deleteStatus       int
		callHistoryAndMeth bool
		callVerifyBefore   bool
		expectFailed       bool
		expectErrContains  string
	}{
		{
			name:               "success-no-session",
			times:              2,
			historyCalls:       2,
			verifyStatus:       http.StatusOK,
			deleteStatus:       http.StatusOK,
			callHistoryAndMeth: true,
			callVerifyBefore:   true,
		},
		{
			name:             "success-with-session",
			session:          "A",
			times:            1,
			historyCalls:     1,
			verifyStatus:     http.StatusOK,
			deleteStatus:     http.StatusOK,
			callVerifyBefore: true,
		},
		{
			name:              "times-mismatch",
			times:             2,
			historyCalls:      1,
			verifyStatus:      http.StatusBadRequest,
			verifyBody:        `{"message":"bad count"}`,
			deleteStatus:      http.StatusOK,
			expectFailed:      true,
			expectErrContains: "expected 2 calls for svc/M",
		},
		{
			name:              "close-error",
			times:             1,
			historyCalls:      1,
			verifyStatus:      http.StatusOK,
			deleteStatus:      http.StatusInternalServerError,
			expectFailed:      true,
			expectErrContains: "batch delete stubs failed with status 500",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			grpcAddr := testkit.StartHealthGRPC(t)
			ts := newManualCleanupT(t.Context())

			var (
				added      atomic.Int32
				history    atomic.Int32
				deleted    atomic.Int32
				listed     atomic.Int32
				createdID  string
				deletedIDs []string
				mu         sync.Mutex
			)

			rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.session != "" {
					require.Equal(t, tc.session, r.Header.Get("X-Gripmock-Session"))
				}

				switch r.URL.Path {
				case "/api/stubs":
					if r.Method == http.MethodPost {
						added.Add(1)
						var payload []map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
						require.Len(t, payload, 1)
						id, _ := payload[0]["id"].(string)
						mu.Lock()
						createdID = id
						mu.Unlock()
						w.WriteHeader(http.StatusOK)
						return
					}

					listed.Add(1)
					w.WriteHeader(http.StatusOK)
					require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{}))
				case "/api/history":
					history.Add(1)
					mu.Lock()
					id := createdID
					mu.Unlock()

					recs := make([]map[string]any, 0, tc.historyCalls)
					for i := 0; i < tc.historyCalls; i++ {
						recs = append(recs, map[string]any{"service": "svc", "method": "M", "stubId": id})
					}
					w.WriteHeader(http.StatusOK)
					require.NoError(t, json.NewEncoder(w).Encode(recs))
				case "/api/stubs/batchDelete":
					deleted.Add(1)
					var ids []string
					require.NoError(t, json.NewDecoder(r.Body).Decode(&ids))
					mu.Lock()
					deletedIDs = append([]string(nil), ids...)
					mu.Unlock()
					w.WriteHeader(tc.deleteStatus)
				case "/api/verify":
					w.WriteHeader(tc.verifyStatus)
					if tc.verifyBody != "" {
						_, _ = w.Write([]byte(tc.verifyBody))
					}
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			})

			opts := []Option{WithRemote(grpcAddr, rest.URL), WithHealthCheckTimeout(time.Second)}
			if tc.session != "" {
				opts = append(opts, WithSession(tc.session))
			}

			mock, err := Run(ts, opts...)
			require.NoError(t, err)

			mock.Stub(By("/svc/M")).Reply(Data("ok", true)).Times(tc.times).Commit()

			if tc.callHistoryAndMeth {
				require.Equal(t, tc.historyCalls, mock.History().Count())
				mock.Verify().Method(By("/svc/M")).Called(ts, tc.times)
			}

			if tc.callVerifyBefore {
				require.NoError(t, mock.Verify().VerifyStubTimesErr())
			}

			ts.RunCleanups()

			require.Equal(t, int32(1), added.Load())
			if tc.callHistoryAndMeth {
				require.GreaterOrEqual(t, history.Load(), int32(1))
			}
			require.Equal(t, int32(1), deleted.Load())
			require.Equal(t, int32(0), listed.Load())

			if tc.expectFailed {
				require.GreaterOrEqual(t, ts.Failed(), 1)
				require.NotEmpty(t, ts.Errors())
				require.Contains(t, ts.FirstError(), tc.expectErrContains)
			} else {
				require.Zero(t, ts.Failed())
				require.Empty(t, ts.Errors())
			}

			mu.Lock()
			require.Len(t, deletedIDs, 1)
			require.Equal(t, createdID, deletedIDs[0])
			mu.Unlock()
		})
	}
}

func TestRunRemoteFullyMockedViaSDK(t *testing.T) {
	t.Parallel()

	// Arrange: minimal gRPC endpoint for Run(...WithRemote...) health checks.
	grpcAddr := testkit.StartHealthGRPC(t)
	ts := newManualCleanupT(t.Context())

	var (
		stubsPostCalls  atomic.Int32
		historyCalls    atomic.Int32
		verifyCalls     atomic.Int32
		batchDeleteCall atomic.Int32

		mu                 sync.Mutex
		createdStubID      string
		verifyExpectedSeen int
		deletedIDs         []string
	)

	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/stubs":
			require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))

			if r.Method == http.MethodPost {
				stubsPostCalls.Add(1)

				var payload []map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				require.Len(t, payload, 1)

				idAny, ok := payload[0]["id"]
				require.True(t, ok)
				id, ok := idAny.(string)
				require.True(t, ok)

				mu.Lock()
				createdStubID = id
				mu.Unlock()

				w.WriteHeader(http.StatusOK)
				return
			}

			require.Equal(t, http.MethodGet, r.Method)

			mu.Lock()
			id := createdStubID
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{"id": id, "session": "A"}}))

		case "/api/history":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
			historyCalls.Add(1)

			mu.Lock()
			id := createdStubID
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{
				{"service": "svc", "method": "M", "request": map[string]any{"x": 1}, "response": map[string]any{"ok": true}, "stubId": id},
				{"service": "svc", "method": "M", "request": map[string]any{"x": 2}, "response": map[string]any{"ok": true}, "stubId": id},
			}))

		case "/api/verify":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
			verifyCalls.Add(1)

			var req struct {
				ExpectedCount int    `json:"expectedCount"`
				Service       string `json:"service"`
				Method        string `json:"method"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Equal(t, "svc", req.Service)
			require.Equal(t, "M", req.Method)

			mu.Lock()
			verifyExpectedSeen = req.ExpectedCount
			mu.Unlock()

			w.WriteHeader(http.StatusOK)

		case "/api/stubs/batchDelete":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "A", r.Header.Get("X-Gripmock-Session"))
			batchDeleteCall.Add(1)

			var ids []string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&ids))
			mu.Lock()
			deletedIDs = append([]string(nil), ids...)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	t.Run("sdk-remote-lifecycle", func(t *testing.T) {
		mock, err := Run(ts,
			WithRemote(grpcAddr, rest.URL),
			WithSession("A"),
			WithHealthCheckTimeout(2*time.Second),
		)
		require.NoError(t, err)

		mock.Stub(By("/svc/M")).
			When(Equals("x", 1)).
			Reply(Data("ok", true)).
			Times(2).
			Commit()

		h := mock.History()
		require.Equal(t, 2, h.Count())
		require.Len(t, h.FilterByMethod("svc", "M"), 2)

		mock.Verify().Method(By("/svc/M")).Called(t, 2)
		mock.Verify().Total(t, 2)
		require.NoError(t, mock.Verify().VerifyStubTimesErr())
	})

	ts.RunCleanups()

	// Assert after subtest and manual cleanup.
	require.Equal(t, int32(1), stubsPostCalls.Load())
	require.GreaterOrEqual(t, historyCalls.Load(), int32(3))
	require.Equal(t, int32(3), verifyCalls.Load())
	require.GreaterOrEqual(t, batchDeleteCall.Load(), int32(1))

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 2, verifyExpectedSeen)
	require.NotEmpty(t, createdStubID)
	require.Equal(t, []string{createdStubID}, deletedIDs)
}

func TestRunRemoteWithDescriptorsUploadsDescriptorSet(t *testing.T) {
	t.Parallel()

	// Arrange
	grpcAddr := testkit.StartHealthGRPC(t)
	ts := newManualCleanupT(t.Context())

	var (
		added             atomic.Int32
		history           atomic.Int32
		deleted           atomic.Int32
		descriptorUploads atomic.Int32
		stubID            string
	)

	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/descriptors":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
			descriptorUploads.Add(1)

			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var fds descriptorpb.FileDescriptorSet
			require.NoError(t, proto.Unmarshal(raw, &fds))
			require.Len(t, fds.File, 1)
			require.Equal(t, "svc.proto", fds.File[0].GetName())

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"ok"}`))

		case "/api/stubs":
			if r.Method == http.MethodPost {
				added.Add(1)
				var payload []map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				require.Len(t, payload, 1)
				stubID, _ = payload[0]["id"].(string)
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{}))
		case "/api/history":
			history.Add(1)
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{"service": "svc", "method": "M", "stubId": stubID}}))
		case "/api/stubs/batchDelete":
			deleted.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/api/verify":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// Act
	name := "svc.proto"
	mock, err := Run(ts,
		WithRemote(grpcAddr, rest.URL),
		WithDescriptors(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: &name}}}),
		WithHealthCheckTimeout(time.Second),
	)
	require.NoError(t, err)

	mock.Stub(By("/svc/M")).Reply(Data("ok", true)).Times(1).Commit()
	require.NoError(t, mock.Verify().VerifyStubTimesErr())

	ts.RunCleanups()

	// Assert
	require.Equal(t, int32(1), descriptorUploads.Load())
	require.Equal(t, int32(1), added.Load())
	require.GreaterOrEqual(t, history.Load(), int32(0))
	require.Equal(t, int32(1), deleted.Load())
	require.Zero(t, ts.Failed())
	require.Empty(t, ts.Errors())
}
