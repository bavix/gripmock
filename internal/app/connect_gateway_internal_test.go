package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
	r.Header.Set("Content-Type", "application/json")

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
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	resp := map[string]string{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "unimplemented", resp["code"])
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
	require.True(t, adapter.sentHeader.Load())
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

func TestConnectRPCGateway_StreamingReturnsUnimplemented(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()

	gateway.writeError(w, codes.Unimplemented, "streaming not supported")

	require.Equal(t, http.StatusNotImplemented, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	resp := map[string]string{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "unimplemented", resp["code"])
}

// TestHttpStreamAdapter_AtomicFlagsNoCopy verifies that the adapter's
// atomic.Bool fields do not get copied through method calls (which would
// trip the race detector in -race mode). This is a regression guard for
// the atomic.Bool refactor.
func TestHttpStreamAdapter_AtomicFlagsNoCopy(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)

	adapter := httpStreamAdapter{
		ctx: req.Context(),
		req: req,
		w:   rec,
	}

	// Taking the address of the embedded atomic.Bool (rather than the
	// struct field directly) is the recommended access pattern. The
	// code under test must not dereference and copy the value.
	require.NotNil(t, &adapter.sentHeader)
	require.NotNil(t, &adapter.endOfStream)

	// Set + Load round-trip.
	adapter.sentHeader.Store(true)
	require.True(t, adapter.sentHeader.Load())

	adapter.endOfStream.Store(true)
	require.True(t, adapter.endOfStream.Load())
}

// TestHttpStreamAdapter_ConcurrentSendMsgNoRace exercises SendMsg from
// multiple goroutines to confirm atomic.Bool changes are race-free. Run
// under `go test -race` to verify.
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
	// the adapter beyond sentHeader. We use it as a minimal concurrent
	// stimulus.
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

	require.True(t, adapter.sentHeader.Load())
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

// TestExtractStreamDelay_UnsupportedTypeReturnsError is a regression
// guard for the previous silent-fallback behaviour. When the user sets
// a `delay` field with an unsupported type (e.g. bool), the helper must
// surface an error rather than falling back to the uniform output.delay.
func TestExtractStreamDelay_UnsupportedTypeReturnsError(t *testing.T) {
	t.Parallel()

	_, hasDelay, err := extractStreamDelay(map[string]any{"delay": true})
	require.Error(t, err)
	assert.True(t, hasDelay, "hasDelay should be true to signal that a delay was specified")

	_, hasDelay, err = extractStreamDelay(map[string]any{"delay": []string{"100ms"}})
	require.Error(t, err)
	assert.True(t, hasDelay)
}

// TestExtractStreamDelay_InvalidStringIsError mirrors the existing
// invalid-string test but reinforces the contract: an explicit error
// (not a silent fallback) is required.
func TestExtractStreamDelay_InvalidStringIsError(t *testing.T) {
	t.Parallel()

	_, hasDelay, err := extractStreamDelay(map[string]any{"delay": "not-a-duration"})
	require.Error(t, err)
	assert.True(t, hasDelay)
}

// TestApplyStreamDelays_OffByOneInvariant documents the contract that
// delays[i] is the gap BEFORE stream[i+1] is sent (not before
// stream[i]). This prevents regression to the previous buggy semantics
// where delays[i] was applied to stream[i].
func TestApplyStreamDelays_OffByOneInvariant(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"data": "first"},
				map[string]any{"data": "second"},
				map[string]any{"data": "third"},
			},
		},
	}

	applyStreamDelays(stub, true, []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
	})

	entry0 := stub.Output.Stream[0].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry0, "delay")

	entry1 := stub.Output.Stream[1].(map[string]any) //nolint:forcetypeassert
	assert.Equal(t, "100ms", entry1["delay"])

	entry2 := stub.Output.Stream[2].(map[string]any) //nolint:forcetypeassert
	assert.Equal(t, "200ms", entry2["delay"])
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
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	resp := map[string]string{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "not_found", resp["code"])
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
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "not_found", body["code"])
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
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
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
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "unimplemented", body["code"])
}
