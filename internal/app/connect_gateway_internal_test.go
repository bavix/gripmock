package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func TestConnectRPCGateway_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/TestService/TestMethod", nil)

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
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
		ctx: nil,
		req: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),
		w:   rec,
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
		ctx: req.Context(),
		req: req,
		w:   nil,
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
		ctx: req.Context(),
		req: req,
		w:   rec,
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
		ctx: req.Context(),
		req: req,
		w:   rec,
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
		ctx: req.Context(),
		req: req,
		w:   rec,
	}

	gateway.handleUnary(mocker, adapter)

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
		ctx: req.Context(),
		req: req,
		w:   rec,
	}

	gateway.handleUnary(mocker, adapter)

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

	gateway.handleWithoutDescriptor(rec, req, "test.Service", "TestMethod")

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

	gateway.handleWithoutDescriptor(rec, req, "test.Service", "TestMethod")

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

	gateway.handleWithoutDescriptor(rec, req, "test.Service", "TestMethod")

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
		ctx:       req.Context(),
		req:       req,
		w:         nil,
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
		ctx:       req.Context(),
		req:       req,
		w:         rec,
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
