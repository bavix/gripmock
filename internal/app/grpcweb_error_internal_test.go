package app

import (
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type grpcWebFrame struct {
	trailer bool
	payload []byte
}

func readGRPCWebFrames(t *testing.T, body []byte) []grpcWebFrame {
	t.Helper()

	frames := make([]grpcWebFrame, 0, 2)

	for len(body) >= ConnectEnvelopeHeaderSize {
		size := binary.BigEndian.Uint32(body[1:5])
		end := ConnectEnvelopeHeaderSize + int(size)
		require.LessOrEqual(t, end, len(body), "frame length runs past the body")

		frames = append(frames, grpcWebFrame{
			trailer: body[0]&grpcwebEnvelopeFlagTrailers != 0,
			payload: body[ConnectEnvelopeHeaderSize:end],
		})
		body = body[end:]
	}

	return frames
}

func newTestGRPCWebAdapter(t *testing.T) (*grpcwebAdapter, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/svc/Method", nil)

	return newGRPCWebAdapter(request, recorder, &grpcMocker{}), recorder
}

func TestGRPCWebErrorIsTrailersOnly(t *testing.T) {
	t.Parallel()

	adapter, recorder := newTestGRPCWebAdapter(t)

	adapter.writeErrorStatus(status.New(codes.NotFound, "no matching stub"))

	frames := readGRPCWebFrames(t, recorder.Body.Bytes())
	require.Len(t, frames, 1, "an error must not be preceded by a data frame")
	require.True(t, frames[0].trailer)
	require.Contains(t, string(frames[0].payload), "grpc-status: 5")
	require.Contains(t, string(frames[0].payload), "grpc-message: no%20matching%20stub")
}

func TestGRPCWebErrorCarriesDetailsInTrailer(t *testing.T) {
	t.Parallel()

	withDetails, err := status.New(codes.InvalidArgument, "bad id").
		WithDetails(&errdetails.ErrorInfo{Reason: "ID_INVALID", Domain: "test.local"})
	require.NoError(t, err)

	adapter, recorder := newTestGRPCWebAdapter(t)

	adapter.writeErrorStatus(withDetails)

	frames := readGRPCWebFrames(t, recorder.Body.Bytes())
	require.Len(t, frames, 1)

	encoded := trailerValue(t, string(frames[0].payload), "grpc-status-details-bin")
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	var restored spb.Status
	require.NoError(t, proto.Unmarshal(raw, &restored))
	require.Equal(t, int32(codes.InvalidArgument), restored.GetCode())
	require.Len(t, restored.GetDetails(), 1)

	info := &errdetails.ErrorInfo{}
	require.NoError(t, restored.GetDetails()[0].UnmarshalTo(info))
	require.Equal(t, "ID_INVALID", info.GetReason())
}

func trailerValue(t *testing.T, trailers, key string) string {
	t.Helper()

	for line := range strings.SplitSeq(trailers, "\r\n") {
		name, value, found := strings.Cut(line, ": ")
		if found && name == key {
			return value
		}
	}

	t.Fatalf("trailer %q not found in %q", key, trailers)

	return ""
}
