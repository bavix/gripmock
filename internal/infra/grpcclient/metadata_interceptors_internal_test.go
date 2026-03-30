package grpcclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type fakeOutgoingClientStream struct {
	contextFn func() context.Context
}

func (f *fakeOutgoingClientStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (f *fakeOutgoingClientStream) Trailer() metadata.MD         { return metadata.MD{} }
func (f *fakeOutgoingClientStream) CloseSend() error             { return nil }
func (f *fakeOutgoingClientStream) SendMsg(any) error            { return nil }
func (f *fakeOutgoingClientStream) RecvMsg(any) error            { return nil }

func (f *fakeOutgoingClientStream) Context() context.Context {
	if f.contextFn != nil {
		return f.contextFn()
	}

	return context.Background()
}

type metadataInterceptorCase struct {
	name        string
	headerName  string
	headerValue string
	unary       grpc.UnaryClientInterceptor
	stream      grpc.StreamClientInterceptor
}

func metadataInterceptorCases() []metadataInterceptorCase {
	return []metadataInterceptorCase{
		{
			name:        "bearer",
			headerName:  "authorization",
			headerValue: "Bearer tkn",
			unary:       UnaryBearerInterceptor("tkn"),
			stream:      StreamBearerInterceptor("tkn"),
		},
		{
			name:        "session",
			headerName:  sessionHeader,
			headerValue: "A",
			unary:       UnarySessionInterceptor("A"),
			stream:      StreamSessionInterceptor("A"),
		},
	}
}

func TestMetadataInterceptorsUnary(t *testing.T) {
	t.Parallel()

	for _, tc := range metadataInterceptorCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runUnaryMetadataCase(t, tc)
		})
	}
}

func TestMetadataInterceptorsStream(t *testing.T) {
	t.Parallel()

	for _, tc := range metadataInterceptorCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runStreamMetadataCase(t, tc)
		})
	}
}

func runUnaryMetadataCase(t *testing.T, tc metadataInterceptorCase) {
	t.Helper()

	ctx := t.Context()
	called := false

	err := tc.unary(
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
			md, ok := metadata.FromOutgoingContext(invCtx)
			require.True(t, ok)
			require.Equal(t, tc.headerValue, md.Get(tc.headerName)[0])

			return nil
		},
	)

	require.NoError(t, err)
	require.True(t, called)
}

func runStreamMetadataCase(t *testing.T, tc metadataInterceptorCase) {
	t.Helper()

	ctx := t.Context()
	called := false

	cs, err := tc.stream(
		ctx,
		&grpc.StreamDesc{},
		nil,
		"/svc/M",
		func(
			streamCtx context.Context,
			_ *grpc.StreamDesc,
			_ *grpc.ClientConn,
			_ string,
			_ ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			called = true
			md, ok := metadata.FromOutgoingContext(streamCtx)
			require.True(t, ok)
			require.Equal(t, tc.headerValue, md.Get(tc.headerName)[0])

			return &fakeOutgoingClientStream{
				contextFn: func() context.Context { return streamCtx },
			}, nil
		},
	)

	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, cs)
}
