package grpcclient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func serverStreamDesc() *grpc.StreamDesc {
	return &grpc.StreamDesc{ServerStreams: true}
}

func invokeWithDesc(
	t *testing.T,
	desc *grpc.StreamDesc,
	timeout time.Duration,
	fs *fakeClientStream,
) (*wrappedClientStream, context.Context) {
	t.Helper()

	streamCtxCh := make(chan context.Context, 1)

	cs, err := StreamTimeoutInterceptor(timeout)(
		t.Context(),
		desc,
		nil,
		"/svc/M",
		func(
			invCtx context.Context,
			_ *grpc.StreamDesc,
			_ *grpc.ClientConn,
			_ string,
			_ ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			streamCtxCh <- invCtx

			fs.contextFn = func() context.Context { return invCtx }

			return fs, nil
		},
	)
	require.NoError(t, err)

	wrapped, ok := cs.(*wrappedClientStream)
	require.True(t, ok)

	return wrapped, <-streamCtxCh
}

func TestStreamTimeoutInterceptorNoDeadlineOnServerStreams(t *testing.T) {
	t.Parallel()

	_, streamCtx := invokeWithDesc(t, serverStreamDesc(), time.Second, &fakeClientStream{})

	_, hasDeadline := streamCtx.Deadline()
	require.False(t, hasDeadline, "server-streaming context must not carry the dial timeout as a deadline")
}

func TestStreamTimeoutInterceptorServerStreamSurvivesTimeout(t *testing.T) {
	t.Parallel()

	const timeout = 100 * time.Millisecond

	wrapped, streamCtx := invokeWithDesc(t, serverStreamDesc(), timeout, &fakeClientStream{})

	require.NoError(t, wrapped.RecvMsg(nil))

	select {
	case <-streamCtx.Done():
		t.Fatal("stream context cancelled after the first message arrived")
	case <-time.After(4 * timeout):
	}

	require.NoError(t, streamCtx.Err())
}

func TestStreamTimeoutInterceptorServerStreamEstablishmentWatchdog(t *testing.T) {
	t.Parallel()

	const timeout = 50 * time.Millisecond

	_, streamCtx := invokeWithDesc(t, serverStreamDesc(), timeout, &fakeClientStream{})

	select {
	case <-streamCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("establishment watchdog did not fire for a silent stream")
	}
}

func TestStreamTimeoutInterceptorHeaderDisarmsWatchdog(t *testing.T) {
	t.Parallel()

	const timeout = 80 * time.Millisecond

	wrapped, streamCtx := invokeWithDesc(t, serverStreamDesc(), timeout, &fakeClientStream{})

	md, err := wrapped.Header()
	require.NoError(t, err)
	require.NotNil(t, md)

	select {
	case <-streamCtx.Done():
		t.Fatal("stream context cancelled after headers were received")
	case <-time.After(3 * timeout):
	}
}

func TestStreamTimeoutInterceptorKeepsDeadlineForUnaryResponseStreams(t *testing.T) {
	t.Parallel()

	_, streamCtx := invokeWithDesc(t, &grpc.StreamDesc{ClientStreams: true}, time.Second, &fakeClientStream{})

	_, hasDeadline := streamCtx.Deadline()
	require.True(t, hasDeadline, "client-streaming context keeps the call deadline")
}
