package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func logUnaryCallWith(t *testing.T, opts LogOptions) string {
	t.Helper()

	var buf bytes.Buffer

	logger := zerolog.New(&buf)
	ctx := logger.WithContext(metadata.NewIncomingContext(t.Context(), metadata.MD{
		"authorization": []string{"Bearer super-secret"},
		"x-tenant":      []string{"acme"},
	}))

	server := &GRPCServer{logOptions: opts}
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Service/Method"}

	req, err := structpb.NewStruct(map[string]any{"value": "abc"})
	require.NoError(t, err)

	resp, err := structpb.NewStruct(map[string]any{"ok": true})
	require.NoError(t, err)

	_, err = server.logUnaryInterceptor(ctx, req, info,
		func(context.Context, any) (any, error) {
			return resp, nil
		})
	require.NoError(t, err)

	return buf.String()
}

func TestLogOptionsDefaultsKeepBodiesAndHideCredentials(t *testing.T) {
	t.Parallel()

	line := logUnaryCallWith(t, DefaultLogOptions())

	require.Contains(t, line, "grpc.request.content")
	require.Contains(t, line, `"value":"abc"`)
	require.Contains(t, line, "grpc.response.content")
	require.Contains(t, line, redactedValue)
	require.NotContains(t, line, "super-secret")
	require.Contains(t, line, "acme")
}

func TestLogOptionsCanDropBodies(t *testing.T) {
	t.Parallel()

	opts := DefaultLogOptions()
	opts.MessageContent = false

	line := logUnaryCallWith(t, opts)

	require.NotContains(t, line, "grpc.request.content")
	require.NotContains(t, line, "grpc.response.content")
	require.Contains(t, line, "grpc.metadata")
}

func TestLogOptionsCanKeepCredentials(t *testing.T) {
	t.Parallel()

	opts := DefaultLogOptions()
	opts.RedactMetadata = false

	line := logUnaryCallWith(t, opts)

	require.Contains(t, line, "super-secret")
	require.NotContains(t, line, redactedValue)
}

func TestLogOptionsHonourConfiguredKeys(t *testing.T) {
	t.Parallel()

	line := logUnaryCallWith(t, LogOptions{RedactMetadata: true, RedactKeys: []string{"x-tenant"}, MessageContent: true})

	require.Contains(t, line, "super-secret")
	require.NotContains(t, line, "acme")
	require.Contains(t, line, redactedValue)
}

func TestLogOptionsStreamRespectsSettings(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := zerolog.New(&buf)
	ctx := logger.WithContext(metadata.NewIncomingContext(t.Context(), metadata.MD{
		"authorization": []string{"Bearer super-secret"},
	}))

	server := &GRPCServer{logOptions: DefaultLogOptions()}
	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Service/Stream"}

	err := server.logStreamInterceptor(nil, contextServerStream{ctx: ctx}, info,
		func(any, grpc.ServerStream) error { return nil })
	require.NoError(t, err)

	line := buf.String()
	require.Contains(t, line, "grpc.metadata")
	require.Contains(t, line, redactedValue)
	require.NotContains(t, line, "super-secret")
	require.Contains(t, line, "grpc.request.content")
}

type contextServerStream struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx
}

func (s contextServerStream) Context() context.Context { return s.ctx }
