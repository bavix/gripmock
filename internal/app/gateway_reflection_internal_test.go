package app

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestIsReflectionMethod(t *testing.T) {
	t.Parallel()

	require.True(t, isReflectionMethod(reflectionServiceV1, reflectionMethodInfo))
	require.True(t, isReflectionMethod(reflectionServiceV1Alpha, reflectionMethodInfo))
	require.False(t, isReflectionMethod(reflectionServiceV1, "Other"))
	require.False(t, isReflectionMethod("helloworld.Greeter", reflectionMethodInfo))
}

func makeConnectFrame(t *testing.T, payload []byte, end bool) []byte {
	t.Helper()

	var flags byte
	if end {
		flags = connectEnvelopeFlagEndStream
	}

	frame := make([]byte, ConnectEnvelopeHeaderSize, ConnectEnvelopeHeaderSize+len(payload))
	frame[0] = flags
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload))) //nolint:gosec

	return append(frame, payload...)
}

func splitConnectFrames(t *testing.T, body []byte) [][]byte {
	t.Helper()

	var frames [][]byte

	for len(body) >= ConnectEnvelopeHeaderSize {
		size := int(binary.BigEndian.Uint32(body[1:5]))
		require.LessOrEqual(t, ConnectEnvelopeHeaderSize+size, len(body))
		frames = append(frames, body[ConnectEnvelopeHeaderSize:ConnectEnvelopeHeaderSize+size])
		body = body[ConnectEnvelopeHeaderSize+size:]
	}

	return frames
}

func TestGatewayReflectionAnswersOverConnect(t *testing.T) {
	t.Parallel()

	registry := descriptors.NewRegistry()
	registerMultiverseDescriptors(t, t.Context(), registry)

	gateway := NewConnectRPCGateway(t.Context(), stuber.NewBudgerigar(), registry, nil, nil, nil, nil)
	router := mux.NewRouter()
	router.Handle("/{service:.+}/{method}", gateway).Methods(http.MethodPost)

	payload, err := protojson.Marshal(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: "multiverse.v1.MultiverseService",
		},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/"+reflectionServiceV1+"/"+reflectionMethodInfo,
		bytes.NewReader(makeConnectFrame(t, payload, false)))
	req.Header.Set(headerContentType, contentTypeConnectJSON)

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, contentTypeConnectJSON, rec.Header().Get(headerContentType)) //nolint:testifylint

	frames := splitConnectFrames(t, rec.Body.Bytes())
	require.Len(t, frames, 2, "one response frame plus the end-of-stream frame")

	var resp reflectionv1.ServerReflectionResponse
	require.NoError(t, protojson.Unmarshal(frames[0], &resp))

	files := resp.GetFileDescriptorResponse().GetFileDescriptorProto()
	require.NotEmpty(t, files)
	require.Contains(t, string(files[0]), "MultiverseService")

	require.JSONEq(t, `{}`, string(frames[1]))
}

func TestGatewayReflectionRejectsUnaryContentType(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(t.Context(), stuber.NewBudgerigar(), descriptors.NewRegistry(), nil, nil, nil, nil)
	router := mux.NewRouter()
	router.Handle("/{service:.+}/{method}", gateway).Methods(http.MethodPost)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/"+reflectionServiceV1+"/"+reflectionMethodInfo, nil)
	req.Header.Set(headerContentType, contentTypeJSON)

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestGatewayServiceInfoProviderListsCompiledServices(t *testing.T) {
	t.Parallel()

	registry := descriptors.NewRegistry()
	registerMultiverseDescriptors(t, t.Context(), registry)

	info := (&gatewayServiceInfoProvider{registry: registry}).GetServiceInfo()

	require.Contains(t, info, "multiverse.v1.MultiverseService")
	require.Contains(t, info, reflectionServiceV1)
}
