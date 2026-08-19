package proxycapture_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestBuildServerStreamStub(t *testing.T) {
	t.Parallel()

	stub := proxycapture.BuildServerStreamStub(
		"test.Service",
		"ServerStreamMethod",
		"session-srv",
		map[string]any{"request": "data"},
		map[string]any{"req-header": "value"},
		[]any{map[string]any{"stream": 1}, map[string]any{"stream": 2}, map[string]any{"stream": 3}},
		proxycapture.ResponseMetadata{Trailers: map[string]string{"trailer": "value"}},
		nil,
	)

	require.Equal(t, "test.Service", stub.Service)
	require.Equal(t, "ServerStreamMethod", stub.Method)
	require.Len(t, stub.Output.Stream, 3)
}

// A replayed stub must put each captured value back where it arrived: a trailer
// stays a trailer, and a key present in both is not spliced into one string.
func TestCaptureMetadataKeepsHeadersAndTrailersApart(t *testing.T) {
	t.Parallel()

	meta := proxycapture.CaptureMetadata(
		metadata.Pairs("x-channel", "header", "content-type", "application/grpc"),
		metadata.Pairs("x-channel", "trailer", "x-audit", "done", "grpc-status", "0"),
	)

	require.Equal(t, "header", meta.Headers["x-channel"])
	require.Equal(t, "trailer", meta.Trailers["x-channel"])
	require.Equal(t, "done", meta.Trailers["x-audit"])
	require.NotContains(t, meta.Trailers, "grpc-status",
		"gRPC generates the status for every call; replaying it would fight the stub")
}

func TestCaptureMetadataEmpty(t *testing.T) {
	t.Parallel()

	meta := proxycapture.CaptureMetadata(nil, nil)

	require.Nil(t, meta.Headers)
	require.Nil(t, meta.Trailers)
}

// A captured stub describes the upstream's answer, not the transport that carried
// it: headers the gRPC stack sets on every response have no place in it.
func TestCaptureMetadataDropsTransportHeaders(t *testing.T) {
	t.Parallel()

	captured := proxycapture.CaptureMetadata(
		metadata.MD{
			"content-type":   []string{"application/grpc"},
			"grpc-encoding":  []string{"gzip"},
			"x-upstream-hdr": []string{"from-upstream"},
		},
		metadata.MD{
			"grpc-status":        []string{"0"},
			"x-upstream-trailer": []string{"tail"},
		},
	)

	require.Equal(t, map[string]string{"x-upstream-hdr": "from-upstream"}, captured.Headers)
	require.Equal(t, map[string]string{"x-upstream-trailer": "tail"}, captured.Trailers)
}

// A captured unary call is a replayable stub: it matches on exactly the request
// that produced it and answers with exactly what came back.
func TestBuildUnaryStubRoundTripsTheCall(t *testing.T) {
	t.Parallel()

	stub := proxycapture.BuildUnaryStub(
		"test.Service", "Unary", "session-a",
		map[string]any{"id": "42"},
		map[string]any{"x-tenant": "acme"},
		map[string]any{"name": "answer"},
		proxycapture.ResponseMetadata{
			Headers:  map[string]string{"x-hdr": "h"},
			Trailers: map[string]string{"x-tail": "t"},
		},
		nil,
	)

	require.Equal(t, "session-a", stub.Session)
	require.Equal(t, stuber.SourceProxy, stub.Source)
	require.Equal(t, map[string]any{"id": "42"}, stub.Input.Equals)
	require.Equal(t, map[string]any{"x-tenant": "acme"}, stub.Headers.Equals)
	require.Equal(t, map[string]any{"name": "answer"}, stub.Output.Data)
	require.Equal(t, map[string]string{"x-hdr": "h"}, stub.Output.Headers)
	require.Equal(t, map[string]string{"x-tail": "t"}, stub.Output.Trailers)
	require.Nil(t, stub.Output.Code)
	require.Empty(t, stub.Output.Error)
}

// An upstream failure is captured as the failure, not as a half-filled success:
// the body it never sent must not be left on the stub.
func TestBuildUnaryStubCapturesStatusAndDropsBody(t *testing.T) {
	t.Parallel()

	details, err := status.New(codes.InvalidArgument, "bad id").
		WithDetails(&errdetails.ErrorInfo{
			Reason: "ID_INVALID",
			Domain: "test.local",
		})
	require.NoError(t, err)

	stub := proxycapture.BuildUnaryStub(
		"test.Service", "Unary", "",
		map[string]any{"id": ""}, nil,
		map[string]any{"name": "never sent"},
		proxycapture.ResponseMetadata{},
		details.Err(),
	)

	require.NotNil(t, stub.Output.Code)
	require.Equal(t, codes.InvalidArgument, *stub.Output.Code)
	require.Equal(t, "bad id", stub.Output.Error)
	require.Nil(t, stub.Output.Data, "a failed call carries no body")

	require.Len(t, stub.Output.Details, 1)
	require.Equal(t, "type.googleapis.com/google.rpc.ErrorInfo", stub.Output.Details[0]["type"],
		"the type URL is what lets a replayed stub rebuild the detail")
	require.Equal(t, "ID_INVALID", stub.Output.Details[0]["reason"])
}

// A client stream is matched message by message, so each captured request keeps
// its own position.
func TestBuildClientStreamStubKeepsMessageOrder(t *testing.T) {
	t.Parallel()

	stub := proxycapture.BuildClientStreamStub(
		"test.Service", "ClientStream", "",
		[]map[string]any{{"n": 1}, {"n": 2}},
		nil,
		map[string]any{"total": 3},
		proxycapture.ResponseMetadata{},
		nil,
	)

	require.Len(t, stub.Inputs, 2)
	require.Equal(t, map[string]any{"n": 1}, stub.Inputs[0].Equals)
	require.Equal(t, map[string]any{"n": 2}, stub.Inputs[1].Equals)
	require.Equal(t, map[string]any{"total": 3}, stub.Output.Data)
}
