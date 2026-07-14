package app

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// extractPayload

func TestExtractPayload_RawBody(t *testing.T) {
	t.Parallel()

	// Raw proto — no frame header.
	raw := []byte{0x0a, 0x04, 0x74, 0x65, 0x73, 0x74}
	got, err := extractPayload(raw)
	require.NoError(t, err)
	require.Equal(t, raw, got)
}

func TestExtractPayload_EmptyBody(t *testing.T) {
	t.Parallel()

	got, err := extractPayload(nil)
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = extractPayload([]byte{})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestExtractPayload_ShortBody(t *testing.T) {
	t.Parallel()

	// Less than 5 bytes — no frame header possible.
	raw := []byte{0x01, 0x02}
	got, err := extractPayload(raw)
	require.NoError(t, err)
	require.Equal(t, raw, got)
}

func TestExtractPayload_UncompressedFrame(t *testing.T) {
	t.Parallel()

	// Properly framed: [0x00][4-byte BE len][data]
	data := []byte{0x0a, 0x04, 0x74, 0x65, 0x73, 0x74}
	frame := buildFrame(0x00, data)

	got, err := extractPayload(frame)
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestExtractPayload_CompressedFrame(t *testing.T) {
	t.Parallel()

	data := []byte{0x01, 0x02, 0x03}
	frame := buildFrame(0x01, data)

	_, err := extractPayload(frame)
	require.Error(t, err)
	require.Contains(t, err.Error(), "compression")
}

func TestExtractPayload_FrameLengthMismatch(t *testing.T) {
	t.Parallel()

	// Frame with correct header but extra trailing bytes — treated as raw.
	data := []byte{0x0a, 0x04, 0x74, 0x65, 0x73, 0x74}
	frame := buildFrame(0x00, data)
	frame = append(frame, 0xde, 0xad) // extra junk

	got, err := extractPayload(frame)
	require.NoError(t, err)
	require.Equal(t, frame, got) // returned as-is
}

func TestExtractPayload_UnknownFlag(t *testing.T) {
	t.Parallel()

	// Flag 0x02 (or any non-0, non-1) with matching length → raw
	data := []byte{0x01}
	frame := buildFrame(0x02, data)

	got, err := extractPayload(frame)
	require.NoError(t, err)
	require.Equal(t, frame, got)
}

// writeGRPCWebTrailers / percentEncode

func TestWriteGRPCWebTrailers_Success(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	writeGRPCWebTrailers(w, codes.OK, "")

	body := w.Body.Bytes()
	require.GreaterOrEqual(t, len(body), 5)

	// flag = 0x80 (trailers)
	require.Equal(t, byte(0x80), body[0], "expected trailers flag 0x80")

	// Should contain grpc-status: 0 but NOT grpc-message
	require.Contains(t, string(body), "grpc-status: 0")
	require.NotContains(t, string(body), "grpc-message")
}

func TestWriteGRPCWebTrailers_Error(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	writeGRPCWebTrailers(w, codes.NotFound, "test message")

	body := w.Body.Bytes()
	require.GreaterOrEqual(t, len(body), 5)

	require.Equal(t, byte(0x80), body[0])
	require.Contains(t, string(body), "grpc-status: 5")
	require.Contains(t, string(body), "grpc-message: test%20message")
}

func TestWriteGRPCWebTrailers_PercentEncodedMessage(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	writeGRPCWebTrailers(w, codes.Internal, "error: \n\r\t\x00")

	body := w.Body.Bytes()
	// Colon : is in unreserved range (0x21-0x7E) — not encoded.
	// Spaces and control chars are encoded.
	require.Contains(t, string(body), "grpc-message: error:%20%0A%0D%09%00")
}

func TestPercentEncode_Spaces(t *testing.T) {
	t.Parallel()

	require.Equal(t, "hello%20world", percentEncode("hello world"))
}

func TestPercentEncode_Noop(t *testing.T) {
	t.Parallel()

	require.Equal(t, "ok123", percentEncode("ok123"))
}

func TestPercentEncode_SpecialChars(t *testing.T) {
	t.Parallel()

	require.Equal(t, "a%25b%0Ac", percentEncode("a%b\nc"))
}

func TestPercentEncode_Empty(t *testing.T) {
	t.Parallel()

	require.Empty(t, percentEncode(""))
}

// isGRPCWebJSONContentType

func TestIsGRPCWebJSONContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ct  string
		yes bool
	}{
		{"application/json", true},
		{grpcwebContentTypeJSON, true},
		{grpcwebContentTypeProto, false},
		{"application/proto", false},
		{"application/grpc-web", false},
		{"application/connect+json", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.ct, func(t *testing.T) {
			t.Parallel()

			got := isGRPCWebJSONContentType(tc.ct)
			require.Equal(t, tc.yes, got)
		})
	}
}

// GRPCWebGateway — method-not-allowed / method-not-found

func TestGRPCWebGateway_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCWebGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/TestService/TestMethod", nil)

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestGRPCWebGateway_MethodNotFound(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCWebGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/UnknownService/UnknownMethod", nil)

	gateway.ServeHTTP(w, r)

	// gRPC-Web always returns 200; error is in trailers.
	body := w.Body.Bytes()
	require.Equal(t, http.StatusOK, w.Code, "gRPC-Web always uses 200")
	require.GreaterOrEqual(t, len(body), 5)

	// First byte must be trailers flag (0x80) — no data, just error trailers.
	require.Equal(t, byte(0x80), body[0], "expected trailers-only response")
	require.Contains(t, string(body), "grpc-status: 5")
	require.Contains(t, string(body), "grpc-message: method%20not%20found")
}

func TestGRPCWebGateway_StubNotFoundWithoutDescriptor(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCWebGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/UnknownService/UnknownMethod",
		bytes.NewReader([]byte(`{}`)))
	r.Header.Set("Content-Type", grpcwebContentTypeJSON)

	gateway.ServeHTTP(w, r)

	body := w.Body.Bytes()
	require.Equal(t, http.StatusOK, w.Code)
	require.GreaterOrEqual(t, len(body), 5)
	require.Contains(t, string(body), "grpc-status")
}

// GRPCWebGateway — via router (mux.Vars)

func TestGRPCWebGateway_RoutedRequest(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCWebGateway(nil, nil, nil, nil, nil, nil)

	router := mux.NewRouter()
	router.Handle("/{service}/{method}", gateway).Methods(http.MethodPost)

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/SomeService/SomeMethod",
		bytes.NewReader([]byte(`{}`)),
	)
	r.Header.Set("Content-Type", grpcwebContentTypeJSON)

	router.ServeHTTP(w, r)

	// Without descriptors, should return error trailers.
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.Bytes()
	require.GreaterOrEqual(t, len(body), 5)
	require.Equal(t, byte(0x80), body[0])
	require.Contains(t, string(body), "grpc-status")
}

// MultiProtocolGateway — dispatch

func TestMultiProtocolGateway_ConnectRPCContentType(t *testing.T) {
	t.Parallel()

	gateway := NewMultiProtocolGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/Svc/Method", nil)
	r.Header.Set("Content-Type", "application/json")

	gateway.ServeHTTP(w, r)

	// ConnectRPC with no descriptors + no budgerigar → method not found → 404
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMultiProtocolGateway_GRPCWebContentType(t *testing.T) {
	t.Parallel()

	gateway := NewMultiProtocolGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/Svc/Method", nil)
	r.Header.Set("Content-Type", grpcwebContentTypeProto)

	gateway.ServeHTTP(w, r)

	// gRPC-Web → always 200 with error trailers
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.Bytes()
	require.GreaterOrEqual(t, len(body), 5)
	require.Equal(t, byte(0x80), body[0])
	require.Contains(t, string(body), "grpc-status")
}

func TestMultiProtocolGateway_GRPCWebTextRejected(t *testing.T) {
	t.Parallel()

	gateway := NewMultiProtocolGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/Svc/Method", nil)
	r.Header.Set("Content-Type", "application/grpc-web-text+proto")

	gateway.ServeHTTP(w, r)

	// gRPC-web-text → rejected with unimplemented
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.Bytes()
	require.Contains(t, string(body), "grpc-status: 12") // codes.Unimplemented
	require.Contains(t, string(body), "grpc-web-text")
}

func TestMultiProtocolGateway_RejectsNonPost(t *testing.T) {
	t.Parallel()

	gateway := NewMultiProtocolGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/Svc/Method", nil)

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// Helpers

// buildFrame creates a gRPC-Web frame: [flag][4-byte BE len][data].
func buildFrame(flag byte, data []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(flag)

	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(data))) //nolint:gosec
	buf.Write(lenBytes)
	buf.Write(data)

	return buf.Bytes()
}
