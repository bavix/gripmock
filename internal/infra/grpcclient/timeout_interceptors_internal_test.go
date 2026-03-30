package grpcclient

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var errStreamCreateFailed = errors.New("stream create failed")

type fakeClientStream struct {
	contextFn func() context.Context
	recvErr   error
	closeErr  error
}

func (f *fakeClientStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (f *fakeClientStream) Trailer() metadata.MD         { return metadata.MD{} }
func (f *fakeClientStream) CloseSend() error             { return f.closeErr }
func (f *fakeClientStream) SendMsg(any) error            { return nil }
func (f *fakeClientStream) RecvMsg(any) error            { return f.recvErr }

func (f *fakeClientStream) Context() context.Context {
	if f.contextFn != nil {
		return f.contextFn()
	}

	return context.Background()
}

func TestUnaryTimeoutInterceptor(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	called := false

	err := UnaryTimeoutInterceptor(500*time.Millisecond)(
		ctx,
		"/svc/M",
		nil,
		nil,
		nil,
		func(
			invCtx context.Context,
			_ string,
			_, _ any,
			_ *grpc.ClientConn,
			_ ...grpc.CallOption,
		) error {
			called = true
			_, ok := invCtx.Deadline()
			require.True(t, ok)

			return nil
		},
	)

	require.NoError(t, err)
	require.True(t, called)
}

func TestStreamTimeoutInterceptorDoesNotCancelImmediately(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fs := &fakeClientStream{}

	streamCtxCh := make(chan context.Context, 1)

	cs, err := StreamTimeoutInterceptor(time.Second)(
		ctx,
		&grpc.StreamDesc{},
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

	streamCtx := <-streamCtxCh
	require.NotNil(t, streamCtx)

	select {
	case <-streamCtx.Done():
		t.Fatal("stream context canceled immediately")
	default:
	}

	require.NoError(t, cs.CloseSend())

	select {
	case <-streamCtx.Done():
	case <-time.After(300 * time.Millisecond):
		t.Fatal("stream context was not canceled on CloseSend")
	}
}

func TestStreamTimeoutInterceptorCancelsOnRecvError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fs := &fakeClientStream{recvErr: io.EOF}

	streamCtxCh := make(chan context.Context, 1)

	cs, err := StreamTimeoutInterceptor(time.Second)(
		ctx,
		&grpc.StreamDesc{},
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

	streamCtx := <-streamCtxCh

	err = cs.RecvMsg(nil)
	require.ErrorIs(t, err, io.EOF)

	select {
	case <-streamCtx.Done():
	case <-time.After(300 * time.Millisecond):
		t.Fatal("stream context was not canceled after RecvMsg error")
	}
}

func TestStreamTimeoutInterceptorCancelsWhenStreamerFails(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	streamCtxCh := make(chan context.Context, 1)

	_, err := StreamTimeoutInterceptor(time.Second)(
		ctx,
		&grpc.StreamDesc{},
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

			return nil, errStreamCreateFailed
		},
	)

	streamCtx := <-streamCtxCh

	require.ErrorIs(t, err, errStreamCreateFailed)
	require.NotNil(t, streamCtx)

	select {
	case <-streamCtx.Done():
	case <-time.After(300 * time.Millisecond):
		t.Fatal("stream context was not canceled when streamer failed")
	}
}
