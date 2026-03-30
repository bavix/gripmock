package sdk

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestStubBuilderReplyHeadersAndIgnoreArrayOrder(t *testing.T) {
	t.Parallel()

	// Arrange
	var got *stuber.Stub
	b := &stubBuilderCore{
		service: "svc",
		method:  "M",
		onCommit: func(stub *stuber.Stub) {
			got = stub
		},
	}

	// Act
	b.When(Equals("items", []any{1, 2})).
		WhenStream(Equals("a", 1), Equals("b", 2)).
		IgnoreArrayOrder().
		Reply(Data("ok", true)).
		ReplyHeaders(map[string]string{"x-a": "1"}).
		ReplyHeaderPairs("x-b", "2").
		Delay(0).
		Priority(7).
		Times(2).
		Commit()

	// Assert
	require.NotNil(t, got)
	require.Equal(t, 2, got.Options.Times)
	require.Equal(t, 7, got.Priority)
	require.Equal(t, "1", got.Output.Headers["x-a"])
	require.Equal(t, "2", got.Output.Headers["x-b"])
	require.True(t, got.Input.IgnoreArrayOrder)
	require.Len(t, got.Inputs, 2)
	require.True(t, got.Inputs[0].IgnoreArrayOrder)
	require.True(t, got.Inputs[1].IgnoreArrayOrder)
}

func TestMapAndHeaderMapHelpers(t *testing.T) {
	t.Parallel()

	// Act
	emptyMap := Map()
	nonEmptyMap := Map("id", "x", "n", 2)
	emptyHeaders := HeaderMap()
	nonEmptyHeaders := HeaderMap("x-id", "abc")

	// Assert
	require.Empty(t, emptyMap.Equals)
	require.Equal(t, "x", nonEmptyMap.Equals["id"])
	require.Equal(t, 2, nonEmptyMap.Equals["n"])
	require.Empty(t, emptyHeaders.Equals)
	require.Equal(t, "abc", nonEmptyHeaders.Equals["x-id"])
}

func TestReplyErrHelpers(t *testing.T) {
	t.Parallel()

	// Act
	errOut := ReplyErr(codes.NotFound, "missing")
	errWithDetails := ReplyErrWithDetails(codes.InvalidArgument, "bad", map[string]any{"type": "x"})

	// Assert
	require.Equal(t, "missing", errOut.Error)
	require.NotNil(t, errOut.Code)
	require.Equal(t, codes.NotFound, *errOut.Code)
	require.Len(t, errWithDetails.Details, 1)
	require.Equal(t, "x", errWithDetails.Details[0]["type"])
}

func TestVerifierDirectBranches(t *testing.T) {
	t.Parallel()

	// Arrange
	recorder := history.NewMemoryStore(0)
	recorder.Record(history.CallRecord{Service: "svc", Method: "M"})
	expected := &atomic.Int32{}
	expected.Store(2)
	v := &verifier{recorder: recorder, expectedTotal: expected}
	ts := &captureTestingT{ctx: t.Context()}

	// Act
	v.Total(ts, 2)
	v.Method(By("/svc/NeverCalled")).Never(ts)
	v.VerifyStubTimes(ts)
	err := v.VerifyStubTimesErr()

	// Assert
	require.Error(t, err)
	require.GreaterOrEqual(t, ts.Failed(), 2)
}

func TestVerifierVerifyStubTimesErrNoExpectedTotal(t *testing.T) {
	t.Parallel()

	// Arrange
	recorder := history.NewMemoryStore(0)
	v := &verifier{recorder: recorder}

	// Act
	err := v.VerifyStubTimesErr()

	// Assert
	require.NoError(t, err)
}
