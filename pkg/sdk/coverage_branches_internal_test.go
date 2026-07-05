package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestContextHelpersFallbackForEmbeddedVerifierHistory(t *testing.T) {
	t.Parallel()

	rec := history.NewMemoryStore(0)
	rec.Record(history.CallRecord{Service: "svc", Method: "M"})
	v := &verifier{recorder: rec}

	err := VerifyStubTimesErrContext(context.Background(), v)
	require.NoError(t, err)

	h := rec
	all, err := HistoryAllContext(context.Background(), h)
	require.NoError(t, err)
	require.Len(t, all, 1)

	count, err := HistoryCountContext(context.Background(), h)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	filtered, err := HistoryFilterByMethodContext(context.Background(), h, "svc", "M")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
}

func TestMethodVerifierCalledSuccessBranch(t *testing.T) {
	t.Parallel()

	// Arrange
	rec := history.NewMemoryStore(0)
	rec.Record(history.CallRecord{Service: "svc", Method: "M"})
	mv := &methodVerifier{recorder: rec, service: "svc", method: "M"}
	ts := &captureTestingT{ctx: t.Context()}

	// Act
	mv.Called(ts, 1)

	// Assert
	require.Zero(t, ts.Failed())
	require.Empty(t, ts.Errors())
}

func TestRemoteVerifierCalledErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("bad-request", func(t *testing.T) {
		t.Parallel()

		rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/verify" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"bad count"}`))
				return
			}

			w.WriteHeader(http.StatusNotFound)
		})

		m := newRemoteMockForServer(rest, "")
		ts := &captureTestingT{ctx: t.Context()}

		m.Verify().Method(By("/svc/M")).Called(ts, 1)

		require.GreaterOrEqual(t, ts.Failed(), 1)
		require.NotEmpty(t, ts.Errors())
		require.Contains(t, ts.FirstError(), "bad count")
	})

	t.Run("server-error", func(t *testing.T) {
		t.Parallel()

		rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/verify" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		})

		m := newRemoteMockForServer(rest, "")
		ts := &captureTestingT{ctx: t.Context()}

		m.Verify().Method(By("/svc/M")).Called(ts, 1)

		require.GreaterOrEqual(t, ts.Failed(), 1)
		require.NotEmpty(t, ts.Errors())
		require.Contains(t, ts.FirstError(), "verify request failed")
	})
}

func TestRemoteVerifierTotalHistoryFetchError(t *testing.T) {
	t.Parallel()

	// Arrange
	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/history" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	m := newRemoteMockForServer(rest, "")
	ts := &captureTestingT{ctx: t.Context()}

	// Act
	m.Verify().Total(ts, 1)

	// Assert
	require.GreaterOrEqual(t, ts.Failed(), 1)
	require.NotEmpty(t, ts.Errors())
	require.Contains(t, ts.FirstError(), "failed to fetch history")
}

func TestRemoteMethodVerifierCalledUsesOperationError(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{}
	m.setOpErr(context.DeadlineExceeded)
	ts := &captureTestingT{ctx: t.Context()}

	// Act
	m.Verify().Method(By("/svc/M")).Called(ts, 1)

	// Assert
	require.GreaterOrEqual(t, ts.Failed(), 1)
	require.NotEmpty(t, ts.Errors())
	require.Contains(t, ts.FirstError(), "operation failed")
	require.Contains(t, ts.FirstError(), context.DeadlineExceeded.Error())
}

func TestRemoteDeleteOwnedStubsEmptyIDs(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &remoteMock{}

	// Act
	err := m.deleteOwnedStubs()

	// Assert
	require.NoError(t, err)
}

func TestStubDataAndStreamItemBranches(t *testing.T) {
	t.Parallel()

	// Data empty branch
	empty := Data()
	require.Empty(t, empty.Data)

	// StreamItem empty branch
	streamEmpty := StreamItem()
	require.Len(t, streamEmpty.Stream, 1)

	// kvToInput/kvToOutput odd path via recover
	require.Panics(t, func() { _ = kvToInput([]any{"k"}, "test") })
	require.Panics(t, func() { _ = kvToOutput([]any{"k"}, "test") })

	// Non-string key branch
	require.Panics(t, func() { _ = kvToInput([]any{1, "v"}, "test") })
	require.Panics(t, func() { _ = kvToOutput([]any{1, "v"}, "test") })

	// MergeHeaders empty branch
	mergedHeaders := MergeHeaders()
	require.Empty(t, mergedHeaders.Equals)
	require.Empty(t, mergedHeaders.Contains)
	require.Empty(t, mergedHeaders.Matches)

	// MergeOutput nil branches
	out := MergeOutput(stuber.Output{}, ReplyHeader("x", "1"), ReplyDelay(0))
	require.Equal(t, "1", out.Headers["x"])
}

func TestMethodVerifierCalledMismatchBranch(t *testing.T) {
	t.Parallel()

	// Arrange
	rec := history.NewMemoryStore(0)
	rec.Record(history.CallRecord{Service: "svc", Method: "M"})
	mv := &methodVerifier{recorder: rec, service: "svc", method: "M"}
	ts := &captureTestingT{ctx: t.Context()}

	// Act
	mv.Called(ts, 2)

	// Assert
	require.GreaterOrEqual(t, ts.Failed(), 1)
	require.NotEmpty(t, ts.Errors())
	require.Contains(t, ts.FirstError(), "expected svc/M called 2 times")
}

func TestRemoteVerifierVerifyStubTimesBranches(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		m := &remoteMock{}
		ts := &captureTestingT{ctx: t.Context()}

		m.Verify().VerifyStubTimes(ts)

		require.Zero(t, ts.Failed())
		require.Empty(t, ts.Errors())
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		m := &remoteMock{}
		m.setOpErr(context.DeadlineExceeded)
		ts := &captureTestingT{ctx: t.Context()}

		m.Verify().VerifyStubTimes(ts)

		require.GreaterOrEqual(t, ts.Failed(), 1)
		require.NotEmpty(t, ts.Errors())
		require.Contains(t, ts.FirstError(), context.DeadlineExceeded.Error())
	})
}

func TestRemoteVerifierUsesTestingContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	m := newRemoteMockForServer(rest, "")
	m.expectedTotal.Store(1)
	m.expectedByMth = map[string]int{methodKey("svc", "M"): 1}
	ts := &captureTestingT{ctx: ctx}

	m.Verify().Method(By("/svc/M")).Called(ts, 1)
	require.GreaterOrEqual(t, ts.Failed(), 1)
	require.Contains(t, ts.FirstError(), context.Canceled.Error())

	ts = &captureTestingT{ctx: ctx}
	m.Verify().Total(ts, 1)
	require.GreaterOrEqual(t, ts.Failed(), 1)
	require.Contains(t, ts.FirstError(), context.Canceled.Error())

	ts = &captureTestingT{ctx: ctx}
	m.Verify().VerifyStubTimes(ts)
	require.GreaterOrEqual(t, ts.Failed(), 1)
	require.Contains(t, ts.FirstError(), context.Canceled.Error())
}

func TestRemoteContextHelpersUseProvidedContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	m := newRemoteMockForServer(rest, "")
	m.expectedTotal.Store(1)
	m.expectedByMth = map[string]int{methodKey("svc", "M"): 1}

	err := VerifyStubTimesErrContext(ctx, m.Verify())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)

	_, err = HistoryAllContext(ctx, m.History())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)

	_, err = HistoryCountContext(ctx, m.History())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)

	_, err = HistoryFilterByMethodContext(ctx, m.History(), "svc", "M")
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRemoteContextHelpersSuccess(t *testing.T) {
	t.Parallel()

	rest := newRemoteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/history":
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{"service": "svc", "method": "M"}}))
		case "/api/verify":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	m := newRemoteMockForServer(rest, "")
	m.expectedTotal.Store(1)
	m.expectedByMth = map[string]int{methodKey("svc", "M"): 1}

	err := VerifyStubTimesErrContext(context.Background(), m.Verify())
	require.NoError(t, err)

	all, err := HistoryAllContext(context.Background(), m.History())
	require.NoError(t, err)
	require.Len(t, all, 1)

	count, err := HistoryCountContext(context.Background(), m.History())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	filtered, err := HistoryFilterByMethodContext(context.Background(), m.History(), "svc", "M")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
}
