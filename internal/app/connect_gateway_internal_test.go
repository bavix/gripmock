package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func TestConnectRPCGateway_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	// The Connect protocol accepts GET and POST; anything else is refused
	// before routing.
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), method, "/TestService/TestMethod", nil)

		gateway.ServeHTTP(w, r)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code, method)
	}
}

func TestConnectRPCGateway_MethodNotFound(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/UnknownService/UnknownMethod", nil)

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestConnectRPCGateway_StubNotFoundWithoutDescriptor(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/UnknownService/UnknownMethod", bytes.NewReader([]byte(`{}`)))
	r.Header.Set("Content-Type", "application/connect+json")

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "not found")
}

func TestConnectRPCGateway_InvalidJSON(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/TestService/TestMethod", bytes.NewReader([]byte("invalid json")))
	r.Header.Set("Content-Type", "application/json")

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestConnectRPCGateway_WriteError(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()

	gateway.writeError(w, codes.Unimplemented, "streaming not supported")

	require.Equal(t, http.StatusNotImplemented, w.Code)
	require.Equal(t, "application/connect+json", w.Header().Get("Content-Type"))

	var resp struct {
		Code    string           `json:"code"`
		Message string           `json:"message"`
		Details []map[string]any `json:"details"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "unimplemented", resp.Code)
	require.NotNil(t, resp.Details)
}

func TestConnectRPCGateway_IsJSONContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ct  string
		yes bool
	}{
		{"application/json", true},
		{"application/connect+json", true},
		{"application/proto", false},
		{"application/grpc", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.ct, func(t *testing.T) {
			t.Parallel()

			got := isJSONContentType(tc.ct)
			require.Equal(t, tc.yes, got)
		})
	}
}

func TestConnectRPCGateway_NewConnectRPCGateway(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	require.NotNil(t, gateway)
}

func TestHttpStreamAdapter_SendMsg_NonProtoMessage(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: nil,
			req: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),
			w:   rec,
		},
	}

	err := adapter.SendMsg("not a proto")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/proto", rec.Header().Get("Content-Type"))
}

func TestHttpStreamAdapter_RecvMsg_EmptyBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader([]byte{}))
	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   nil,
		},
	}

	_ = adapter.RecvMsg(nil)
}

func TestHttpHeadersToGRPCContext_NoHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	out := httpHeadersToGRPCContext(ctx, http.Header{})

	require.Equal(t, ctx, out, "empty headers should return original context")
}

func TestHttpHeadersToGRPCContext_ExcludesConnectHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hdr := http.Header{}
	hdr.Set("Content-Type", "application/json")
	hdr.Set("Accept-Encoding", "gzip")
	hdr.Set("User-Agent", "test")
	hdr.Set("Connect-Protocol-Version", "1")
	hdr.Set("X-Custom-Header", "value")

	out := httpHeadersToGRPCContext(ctx, hdr)
	md, ok := metadata.FromIncomingContext(out)
	require.True(t, ok)

	require.NotContains(t, md, "content-type")
	require.NotContains(t, md, "accept-encoding")
	require.NotContains(t, md, "user-agent")
	require.NotContains(t, md, "connect-protocol-version")
	require.Equal(t, []string{"value"}, md.Get("x-custom-header"))
}

func TestHttpHeadersToGRPCContext_PropagatesSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hdr := http.Header{}
	hdr.Set("X-Gripmock-Session", "my-session-123")

	out := httpHeadersToGRPCContext(ctx, hdr)
	md, ok := metadata.FromIncomingContext(out)
	require.True(t, ok)
	require.Equal(t, []string{"my-session-123"}, md.Get("x-gripmock-session"))
}

func TestConnectRPCGateway_RoutedRequest_ParsesVars(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)

	router := mux.NewRouter()
	router.Handle("/{service}/{method}", gateway).Methods(http.MethodPost)

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/SomeService/SomeMethod",
		bytes.NewReader([]byte(`{}`)),
	)
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	// Without descriptors, the gateway returns 404 (not 200).
	require.Equal(t, http.StatusNotFound, w.Code)
}

// TestHttpStreamAdapter_AtomicFlagsNoCopy verifies that the adapter's
// endOfStream (still atomic.Bool) must not be copied through method
// calls, otherwise the race detector would fire in -race mode.
func TestHttpStreamAdapter_EndOfStreamNoCopy(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)

	adapter := httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   rec,
		},
	}

	require.NotNil(t, &adapter.endOfStream)

	adapter.endOfStream.Store(true)
	require.True(t, adapter.endOfStream.Load())
}

// TestHttpStreamAdapter_ConcurrentSendMsgNoRace exercises SendMsg from
// multiple goroutines to confirm sendHeaderOnce is race-free.
func TestHttpStreamAdapter_ConcurrentSendMsgNoRace(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(nil))

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   rec,
		},
	}

	// SendMsg with a non-proto message short-circuits without writing to
	// the body. We use it as a minimal concurrent stimulus.
	var wg sync.WaitGroup

	const goroutines = 8

	for i := range goroutines {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := range 50 {
				_ = adapter.SendMsg(id*100 + j)
			}
		}(i)
	}

	wg.Wait()

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/proto", rec.Header().Get("Content-Type"))
}

// TestHttpHeadersToGRPCContext_PreservesAllHeaders verifies that custom
// headers survive the conversion from HTTP headers into gRPC incoming
// metadata. The user-facing contract is: every non-excluded header must
// be available to stub matching.
func TestHttpHeadersToGRPCContext_PreservesAllHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hdr := http.Header{}
	hdr.Set("X-Tenant-Id", "tenant-42")
	hdr.Set("X-Request-Id", "req-abc")
	hdr.Set("Authorization", "Bearer token")

	out := httpHeadersToGRPCContext(ctx, hdr)
	md, ok := metadata.FromIncomingContext(out)

	require.True(t, ok)
	assert.Equal(t, []string{"tenant-42"}, md.Get("x-tenant-id"))
	assert.Equal(t, []string{"req-abc"}, md.Get("x-request-id"))
	assert.Equal(t, []string{"Bearer token"}, md.Get("authorization"))
}

// TestConnectExcludedHeaders_ConnectProtocol verifies that Connect-RPC
// protocol headers are filtered before reaching gRPC stub matching.
// These are transport-layer concerns that should not influence stub
// routing.
func TestConnectExcludedHeaders_ConnectProtocol(t *testing.T) {
	t.Parallel()

	hdr := http.Header{}
	hdr.Set("Connect-Protocol-Version", "1")
	hdr.Set("Connect-Timeout-Ms", "10000")

	got := extractConnectHeaders(hdr)
	assert.NotContains(t, got, "connect-protocol-version")
	assert.NotContains(t, got, "connect-timeout-ms")
}

// TestConnectRPCGateway_HandleUnary_StubNotFound verifies that when no
// stub matches the request, handleUnary writes a 404 error response.
func TestConnectRPCGateway_HandleUnary_StubNotFound(t *testing.T) {
	t.Parallel()

	structDesc := (&structpb.Struct{}).ProtoReflect().Descriptor()
	bg := stuber.NewBudgerigar()
	gateway := NewConnectRPCGateway(bg, nil, nil, nil, nil, nil)

	mocker := &grpcMocker{
		budgerigar:      bg,
		templateEngine:  template.New(t.Context(), nil),
		errorFormatter:  NewErrorFormatter(),
		inputDesc:       structDesc,
		outputDesc:      structDesc,
		fullServiceName: "test.Service",
		methodName:      "TestMethod",
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", bytes.NewReader([]byte(`{"name":"Alice"}`)))
	req.Header.Set("Content-Type", "application/json")

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   rec,
		},
	}

	gateway.handleUnary(mocker, adapter, nil)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))

	var resp struct {
		Code    string           `json:"code"`
		Message string           `json:"message"`
		Details []map[string]any `json:"details"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "not_found", resp.Code)
}

// TestConnectRPCGateway_HandleUnary_Success verifies that handleUnary
// correctly processes a matching stub and returns the expected JSON response.
func TestConnectRPCGateway_HandleUnary_Success(t *testing.T) {
	t.Parallel()

	structDesc := (&structpb.Struct{}).ProtoReflect().Descriptor()
	bg := stuber.NewBudgerigar()

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Data: map[string]any{"name": "Alice"},
		},
	}
	bg.PutMany(stub)

	gateway := NewConnectRPCGateway(bg, nil, nil, nil, nil, nil)

	mocker := &grpcMocker{
		budgerigar:      bg,
		templateEngine:  template.New(t.Context(), nil),
		errorFormatter:  NewErrorFormatter(),
		inputDesc:       structDesc,
		outputDesc:      structDesc,
		fullServiceName: "test.Service",
		methodName:      "TestMethod",
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", bytes.NewReader([]byte(`{"name":"Alice"}`)))
	req.Header.Set("Content-Type", "application/json")

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   rec,
		},
	}

	gateway.handleUnary(mocker, adapter, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Alice", resp["name"])
}

// TestConnectRPCGateway_HandleWithoutDescriptor_StubNotFound verifies
// the fallback path returns 404 when no stub matches.
func TestConnectRPCGateway_HandleWithoutDescriptor_StubNotFound(t *testing.T) {
	t.Parallel()

	bg := stuber.NewBudgerigar()
	gateway := NewConnectRPCGateway(bg, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	gateway.handleWithoutDescriptor(rec, req, "test.Service", "TestMethod", connectResponse{})

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))

	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "not_found", body.Code)
}

// TestConnectRPCGateway_HandleWithoutDescriptor_EmptyData verifies the
// fallback path returns 200 with "{}" when a stub with no output data matches.
func TestConnectRPCGateway_HandleWithoutDescriptor_EmptyData(t *testing.T) {
	t.Parallel()

	bg := stuber.NewBudgerigar()
	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output:  stuber.Output{},
	}
	bg.PutMany(stub)

	gateway := NewConnectRPCGateway(bg, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	gateway.handleWithoutDescriptor(rec, req, "test.Service", "TestMethod", connectResponse{})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))
	require.Equal(t, "{}", rec.Body.String())
}

// TestConnectRPCGateway_HandleWithoutDescriptor_WithData verifies the
// fallback path returns Unimplemented when a stub has output data but no
// proto descriptor is available to encode it.
func TestConnectRPCGateway_HandleWithoutDescriptor_WithData(t *testing.T) {
	t.Parallel()

	bg := stuber.NewBudgerigar()
	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Data: map[string]any{"name": "Alice"},
		},
	}
	bg.PutMany(stub)

	gateway := NewConnectRPCGateway(bg, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	gateway.handleWithoutDescriptor(rec, req, "test.Service", "TestMethod", connectResponse{})

	require.Equal(t, http.StatusNotImplemented, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))

	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "unimplemented", body.Code)
}

// TestHttpStreamAdapter_EndStreamFrameReturnsEOF verifies that recvStreamingMessage
// returns io.EOF when the client sends an endStream-only frame (empty data + endStream
// flag). Previously the frame was decoded as a zero-value message, hiding the end-of-stream
// signal from the handler.
func TestHttpStreamAdapter_EndStreamFrameReturnsEOF(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeConnectFrame(&buf, nil, true))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", "application/connect+json")

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   nil,
		},
		streaming: true,
	}

	// RecvMsg should return io.EOF for a pure endStream envelope.
	msg := &structpb.Struct{}
	err := adapter.RecvMsg(msg)
	require.ErrorIs(t, err, io.EOF)
}

// TestHttpStreamAdapter_SendMsgNotAffectedByClientEndStream verifies that
// the server's SendMsg does NOT set the endStream flag on outbound envelopes
// after receiving a client endStream frame. The client and server end-of-stream
// signals are independent in the Connect RPC protocol.
func TestHttpStreamAdapter_SendMsgNotAffectedByClientEndStream(t *testing.T) {
	t.Parallel()

	var inputBuf bytes.Buffer
	require.NoError(t, writeConnectFrame(&inputBuf, nil, true))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &inputBuf)
	req.Header.Set("Content-Type", "application/connect+json")

	rec := httptest.NewRecorder()
	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   rec,
		},
		streaming: true,
	}

	// Consume the client endStream frame — must return io.EOF.
	err := adapter.RecvMsg(&structpb.Struct{})
	require.ErrorIs(t, err, io.EOF)

	// Send a server response — the envelope must NOT carry endStream flag.
	msg := &structpb.Struct{Fields: map[string]*structpb.Value{
		"key": structpb.NewStringValue("value"),
	}}
	require.NoError(t, adapter.SendMsg(msg))

	// Read back the server envelope and assert endStream is clear.
	frame, err := readConnectFrame(rec.Body)
	require.NoError(t, err)
	require.Zero(t, frame.flags&connectEnvelopeFlagEndStream,
		"server response must not have endStream flag set")
}

func TestConnectAdapterUnaryTrailersUsePrefix(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	adapter := &httpStreamAdapter{baseStreamAdapter: baseStreamAdapter{w: w}}

	adapter.SetTrailer(metadata.Pairs("x-audit", "done"))
	adapter.SetTrailer(metadata.Pairs("Trailer-x-already", "kept"))

	require.Equal(t, []string{"done"}, w.Header().Values("Trailer-X-Audit"),
		"Connect unary carries trailing metadata as Trailer- prefixed headers")
	require.Equal(t, []string{"kept"}, w.Header().Values("Trailer-X-Already"),
		"an already-prefixed key must not be double-prefixed")
	require.Nil(t, adapter.takeTrailerMetadata(), "unary writes headers, it does not buffer")
}

func TestConnectAdapterStreamingTrailersRideEndStream(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	adapter := &httpStreamAdapter{baseStreamAdapter: baseStreamAdapter{w: w}, streaming: true}

	adapter.SetTrailer(metadata.Pairs("x-audit", "done"))

	require.Empty(t, w.Header().Values("Trailer-X-Audit"), "streaming must not use HTTP headers")
	require.Equal(t, map[string][]string{"x-audit": {"done"}}, adapter.takeTrailerMetadata())
}

func TestConnectStreamingErrorIsNestedUnderError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/svc/Method", nil)
	r.Header.Set(headerContentType, contentTypeConnectJSON)
	adapter := &httpStreamAdapter{baseStreamAdapter: baseStreamAdapter{w: w, req: r}, streaming: true}
	adapter.SetTrailer(metadata.Pairs("x-audit", "done"))

	adapter.writeErrorStatus(status.New(codes.NotFound, "missing"))

	body := w.Body.Bytes()
	require.Greater(t, len(body), ConnectEnvelopeHeaderSize)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body[ConnectEnvelopeHeaderSize:], &payload))
	require.Contains(t, payload, "error", "the Connect protocol nests the error in the end-stream envelope")
	require.Contains(t, payload, "metadata")
}

// The Connect protocol pins each code to an HTTP status; canceled maps to
// 499, which net/http has no constant for and which was previously 408.
func TestErrorCodeToHTTPStatusFollowsConnectProtocol(t *testing.T) {
	t.Parallel()

	for code, want := range map[codes.Code]int{
		codes.Canceled:           499,
		codes.Unknown:            http.StatusInternalServerError,
		codes.InvalidArgument:    http.StatusBadRequest,
		codes.DeadlineExceeded:   http.StatusGatewayTimeout,
		codes.NotFound:           http.StatusNotFound,
		codes.AlreadyExists:      http.StatusConflict,
		codes.PermissionDenied:   http.StatusForbidden,
		codes.ResourceExhausted:  http.StatusTooManyRequests,
		codes.FailedPrecondition: http.StatusBadRequest,
		codes.Aborted:            http.StatusConflict,
		codes.OutOfRange:         http.StatusBadRequest,
		codes.Unimplemented:      http.StatusNotImplemented,
		codes.Internal:           http.StatusInternalServerError,
		codes.Unavailable:        http.StatusServiceUnavailable,
		codes.DataLoss:           http.StatusInternalServerError,
		codes.Unauthenticated:    http.StatusUnauthorized,
	} {
		require.Equal(t, want, ErrorCodeToHTTPStatus(code), "code %s", code)
	}
}

func TestConnectRPCGateway_RejectsMalformedTimeout(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/TestService/TestMethod", nil)
	r.Header.Set(headerConnectTimeoutMs, "not-a-number")

	gateway.ServeHTTP(w, r)

	// The method is unknown too, but a malformed timeout must not be ignored:
	// running the call unbounded would defeat the caller's deadline.
	require.NotEqual(t, http.StatusOK, w.Code)
}

// A streaming response that carries no messages never called SendMsg, so the
// content type was never set and Go sniffed it as application/octet-stream.
func TestConnectStreamingEndFrameSetsContentType(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/svc/Method", nil)
	r.Header.Set(headerContentType, contentTypeConnectJSON)

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{w: rec, req: r},
		streaming:         true,
	}

	adapter.sendHeader()

	body, err := json.Marshal(connectEndStream{})
	require.NoError(t, err)
	require.NoError(t, writeConnectFrame(rec, body, true))

	require.Equal(t, contentTypeConnectJSON, rec.Header().Get(headerContentType)) //nolint:testifylint
	require.JSONEq(t, "{}", rec.Body.String()[ConnectEnvelopeHeaderSize:],
		"the end-of-stream envelope must carry a JSON object")
}

func TestIsConnectStreamContentType(t *testing.T) {
	t.Parallel()

	for ct, want := range map[string]bool{
		contentTypeConnectJSON:                true,
		contentTypeConnectProto:               true,
		"application/connect+json; charset=1": true,
		"APPLICATION/CONNECT+JSON":            true,
		contentTypeJSON:                       false,
		contentTypeProto:                      false,
		"application/grpc-web+proto":          false,
		"":                                    false,
	} {
		require.Equal(t, want, isConnectStreamContentType(ct), "content type %q", ct)
	}
}

// The protocol requires application/connect+{codec} for streaming RPCs; the
// unary forms must be answered with 415, not silently accepted.
func TestConnectRPCGateway_StreamingRejectsUnaryContentType(t *testing.T) {
	t.Parallel()

	registry := descriptors.NewRegistry()
	registerMultiverseDescriptors(t, t.Context(), registry)

	gateway := NewConnectRPCGateway(stuber.NewBudgerigar(), registry, nil, nil, nil, nil)
	router := mux.NewRouter()
	router.Handle("/{service:.+}/{method}", gateway).Methods(http.MethodPost)

	for ct, want := range map[string]int{
		contentTypeJSON:        http.StatusUnsupportedMediaType,
		contentTypeProto:       http.StatusUnsupportedMediaType,
		contentTypeConnectJSON: http.StatusOK,
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
			"/multiverse.v1.MultiverseService/StreamData", strings.NewReader(""))
		req.Header.Set(headerContentType, ct)

		router.ServeHTTP(rec, req)

		require.Equal(t, want, rec.Code, "content type %q", ct)
	}
}

func registerMultiverseDescriptors(t *testing.T, ctx context.Context, registry *descriptors.Registry) {
	t.Helper()

	fdsList, err := protoset.Build(ctx, nil,
		[]string{filepath.Join("..", "..", "examples", "projects", "multiverse", "service.proto")}, nil)
	require.NoError(t, err)

	var merged descriptorpb.FileDescriptorSet
	for _, set := range fdsList {
		merged.File = append(merged.File, set.GetFile()...)
	}

	files, err := decodeDescriptorFiles(&merged)
	require.NoError(t, err)

	for _, fd := range files {
		registry.Register(fd)
	}
}

// The protocol makes rejection the server's choice, so the check stays off
// until asked for; when on it must accept the header form and the GET query
// form the spec defines.
func TestConnectRPCGateway_RequireProtocolVersion(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		require bool
		method  string
		target  string
		header  string
		wantErr bool
	}{
		"off by default":     {require: false, method: http.MethodPost, target: "/svc/M"},
		"missing header":     {require: true, method: http.MethodPost, target: "/svc/M", wantErr: true},
		"header present":     {require: true, method: http.MethodPost, target: "/svc/M", header: "1"},
		"wrong header value": {require: true, method: http.MethodPost, target: "/svc/M", header: "2", wantErr: true},
		"get query present":  {require: true, method: http.MethodGet, target: "/svc/M?connect=v1"},
		"get query missing":  {require: true, method: http.MethodGet, target: "/svc/M", wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gateway := NewConnectRPCGateway(stuber.NewBudgerigar(), descriptors.NewRegistry(), nil, nil, nil, nil)
			gateway.RequireProtocolVersion(tc.require)

			router := mux.NewRouter()
			router.Handle("/{service:.+}/{method}", gateway).Methods(http.MethodPost, http.MethodGet)

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.target, nil)

			if tc.header != "" {
				req.Header.Set(headerConnectProtocolVersion, tc.header)
			}

			router.ServeHTTP(rec, req)

			if tc.wantErr {
				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Contains(t, rec.Body.String(), "invalid_argument")

				return
			}

			require.NotEqual(t, http.StatusBadRequest, rec.Code)
		})
	}
}
