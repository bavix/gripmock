package app

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// ── Transport-level test helpers ─────────────────────────────────────────────

// startH2CServer wraps handler with h2c and starts a cleartext test server that
// accepts both HTTP/1.1 and HTTP/2 on the same port, mirroring what
// deps.ConnectServe does in production.
func startH2CServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	srv := httptest.NewUnstartedServer(h2c.NewHandler(handler, &http2.Server{}))
	srv.Start()
	t.Cleanup(srv.Close)

	return srv
}

// http1Client returns a client that speaks HTTP/1.1 only.
// For cleartext connections Go's default transport never upgrades to h2c,
// but we use an explicit zero-value Transport to make the intent unambiguous.
func http1Client() *http.Client {
	return &http.Client{Transport: &http.Transport{}}
}

// h2cClient returns a client that uses HTTP/2 cleartext via the
// "prior knowledge" upgrade path (RFC 7540 §3.4).
// The DialTLSContext field is repurposed: when AllowHTTP is true the http2
// package calls it even for plain TCP connections.
func h2cClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		},
	}
}

// protocolCase parameterises tests that must pass on both HTTP versions.
type protocolCase struct {
	name      string
	client    *http.Client
	wantProto string // expected resp.Proto from the HTTP client
}

func bothProtocols() []protocolCase {
	return []protocolCase{
		{"HTTP/1.1", http1Client(), "HTTP/1.1"},
		{"HTTP/2", h2cClient(), "HTTP/2.0"},
	}
}

// newGreeterConnectHandler builds a ConnectHandler for helloworld.Greeter
// pre-loaded with the supplied stubs.
func newGreeterConnectHandler(t *testing.T, stubs ...*stuber.Stub) *ConnectHandler {
	t.Helper()

	bud := stuber.NewBudgerigar()
	if len(stubs) > 0 {
		bud.PutMany(stubs...)
	}

	return NewConnectHandler(bud, buildGreeterRegistry(t), nil)
}

// ── Unary ─────────────────────────────────────────────────────────────────────

func TestConnectUnary_HTTP1andHTTP2_Success(t *testing.T) {
	t.Parallel()

	for _, tc := range bothProtocols() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &stuber.Stub{
				ID:      uuid.New(),
				Service: "helloworld.Greeter",
				Method:  "SayHello",
				Input:   stuber.InputData{Equals: map[string]any{"name": "World"}},
				Output:  stuber.Output{Data: map[string]any{"message": "Hello, World!"}},
			}
			srv := startH2CServer(t, newGreeterConnectHandler(t, stub))

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				srv.URL+"/helloworld.Greeter/SayHello",
				strings.NewReader(`{"name":"World"}`))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := tc.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, tc.wantProto, resp.Proto, "wrong HTTP version negotiated")

			var body map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			require.Equal(t, "Hello, World!", body["message"])
		})
	}
}

func TestConnectUnary_HTTP1andHTTP2_StubNotFound(t *testing.T) {
	t.Parallel()

	for _, tc := range bothProtocols() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// No stubs registered → handler must return 404 on both protocols.
			srv := startH2CServer(t, newGreeterConnectHandler(t))

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				srv.URL+"/helloworld.Greeter/SayHello",
				strings.NewReader(`{"name":"Ghost"}`))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := tc.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNotFound, resp.StatusCode)
			require.Equal(t, tc.wantProto, resp.Proto)

			var body map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			require.Equal(t, "not_found", body["code"])
		})
	}
}

func TestConnectUnary_HTTP1andHTTP2_CustomResponseHeaders(t *testing.T) {
	t.Parallel()

	for _, tc := range bothProtocols() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &stuber.Stub{
				ID:      uuid.New(),
				Service: "helloworld.Greeter",
				Method:  "SayHello",
				Input:   stuber.InputData{Equals: map[string]any{"name": "World"}},
				Output: stuber.Output{
					Data:    map[string]any{"message": "hi"},
					Headers: map[string]string{"x-custom-header": "from-stub"},
				},
			}
			srv := startH2CServer(t, newGreeterConnectHandler(t, stub))

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				srv.URL+"/helloworld.Greeter/SayHello",
				strings.NewReader(`{"name":"World"}`))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := tc.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, tc.wantProto, resp.Proto)
			require.Equal(t, "from-stub", resp.Header.Get("x-custom-header"))
		})
	}
}

// ── Server streaming ──────────────────────────────────────────────────────────

func TestConnectServerStream_HTTP1andHTTP2_MultipleFrames(t *testing.T) {
	t.Parallel()

	for _, tc := range bothProtocols() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &stuber.Stub{
				ID:      uuid.New(),
				Service: "chat.ChatService",
				Method:  "ReceiveMessages",
				Input:   stuber.InputData{Equals: map[string]any{"user": "alice"}},
				Output: stuber.Output{
					Stream: []any{
						map[string]any{"user": "server", "text": "msg1"},
						map[string]any{"user": "server", "text": "msg2"},
					},
				},
			}
			srv := startH2CServer(t, newChatConnectHandler(t, stub))

			body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "alice"}))

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				srv.URL+"/chat.ChatService/ReceiveMessages", body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/connect+json")

			resp, err := tc.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, tc.wantProto, resp.Proto)
			require.Equal(t, "application/connect+json", resp.Header.Get("Content-Type"))

			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			frames := parseAllFrames(t, raw)
			require.Len(t, frames, 3, "expected 2 data frames + 1 end-stream")

			require.False(t, frames[0].isEndStream)
			var msg0 map[string]any
			require.NoError(t, json.Unmarshal(frames[0].payload, &msg0))
			require.Equal(t, "msg1", msg0["text"])

			require.False(t, frames[1].isEndStream)
			var msg1 map[string]any
			require.NoError(t, json.Unmarshal(frames[1].payload, &msg1))
			require.Equal(t, "msg2", msg1["text"])

			require.True(t, frames[2].isEndStream)
			require.Equal(t, []byte("{}"), frames[2].payload)
		})
	}
}

func TestConnectServerStream_HTTP1andHTTP2_StubNotFound(t *testing.T) {
	t.Parallel()

	for _, tc := range bothProtocols() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := startH2CServer(t, newChatConnectHandler(t)) // no stubs

			body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "nobody"}))

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				srv.URL+"/chat.ChatService/ReceiveMessages", body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/connect+json")

			resp, err := tc.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, tc.wantProto, resp.Proto)

			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			frames := parseAllFrames(t, raw)
			requireErrorEndStream(t, frames, "not_found")
		})
	}
}

// ── Client streaming ──────────────────────────────────────────────────────────

func TestConnectClientStream_HTTP1andHTTP2_MultipleMessages(t *testing.T) {
	t.Parallel()

	for _, tc := range bothProtocols() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &stuber.Stub{
				ID:      uuid.New(),
				Service: "chat.ChatService",
				Method:  "SendMessage",
				Inputs: []stuber.InputData{
					{Equals: map[string]any{"user": "alice", "text": "hello"}},
					{Equals: map[string]any{"user": "alice", "text": "world"}},
				},
				Output: stuber.Output{
					Data: map[string]any{"success": true, "message": "got both"},
				},
			}
			srv := startH2CServer(t, newChatConnectHandler(t, stub))

			body := buildStreamBody(
				jsonDataFrame(t, map[string]any{"user": "alice", "text": "hello"}),
				jsonDataFrame(t, map[string]any{"user": "alice", "text": "world"}),
				jsonEndStreamFrame(t),
			)

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				srv.URL+"/chat.ChatService/SendMessage", body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/connect+json")

			resp, err := tc.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, tc.wantProto, resp.Proto)

			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			frames := parseAllFrames(t, raw)
			require.Len(t, frames, 2, "expected 1 response frame + 1 end-stream")

			require.False(t, frames[0].isEndStream)
			var got map[string]any
			require.NoError(t, json.Unmarshal(frames[0].payload, &got))
			require.Equal(t, true, got["success"])
			require.Equal(t, "got both", got["message"])

			require.True(t, frames[1].isEndStream)
			require.Equal(t, []byte("{}"), frames[1].payload)
		})
	}
}

func TestConnectClientStream_HTTP1andHTTP2_StubNotFound(t *testing.T) {
	t.Parallel()

	for _, tc := range bothProtocols() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := startH2CServer(t, newChatConnectHandler(t)) // no stubs

			body := buildStreamBody(
				jsonDataFrame(t, map[string]any{"user": "nobody", "text": "hi"}),
				jsonEndStreamFrame(t),
			)

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				srv.URL+"/chat.ChatService/SendMessage", body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/connect+json")

			resp, err := tc.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, tc.wantProto, resp.Proto)

			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			frames := parseAllFrames(t, raw)
			requireErrorEndStream(t, frames, "not_found")
		})
	}
}

// ── Bidirectional streaming ───────────────────────────────────────────────────
//
// Bidi streaming requires HTTP/2. HTTP/1.1 is half-duplex: Go's server closes
// r.Body once the handler starts writing a response, making it impossible to
// read subsequent request frames. The Connect protocol spec mandates HTTP/2 for
// bidi streaming; the HTTP/1.1 limitation test below documents this boundary.

func TestConnectBidiStream_HTTP2_SingleExchange(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "Chat",
		Input:   stuber.InputData{Equals: map[string]any{}},
		Output:  stuber.Output{Data: map[string]any{"user": "server", "text": "pong"}},
	}
	srv := startH2CServer(t, newChatConnectHandler(t, stub))

	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping"}),
		jsonEndStreamFrame(t),
	)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		srv.URL+"/chat.ChatService/Chat", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/connect+json")

	resp, err := h2cClient().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "HTTP/2.0", resp.Proto)
	require.Equal(t, "application/connect+json", resp.Header.Get("Content-Type"))

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	frames := parseAllFrames(t, raw)
	require.Len(t, frames, 2, "expected 1 response frame + 1 end-stream")

	require.False(t, frames[0].isEndStream)
	var msg map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &msg))
	require.Equal(t, "pong", msg["text"])
	require.Equal(t, "server", msg["user"])

	require.True(t, frames[1].isEndStream)
	require.Equal(t, []byte("{}"), frames[1].payload)
}

func TestConnectBidiStream_HTTP2_MultipleExchanges(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "Chat",
		Input:   stuber.InputData{Equals: map[string]any{}},
		Output:  stuber.Output{Data: map[string]any{"user": "server", "text": "pong"}},
	}
	srv := startH2CServer(t, newChatConnectHandler(t, stub))

	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping1"}),
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping2"}),
		jsonEndStreamFrame(t),
	)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		srv.URL+"/chat.ChatService/Chat", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/connect+json")

	resp, err := h2cClient().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "HTTP/2.0", resp.Proto)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	frames := parseAllFrames(t, raw)
	require.Len(t, frames, 3, "expected 2 response frames + 1 end-stream")
	require.False(t, frames[0].isEndStream)
	require.False(t, frames[1].isEndStream)
	require.True(t, frames[2].isEndStream)

	for i := range 2 {
		var msg map[string]any
		require.NoError(t, json.Unmarshal(frames[i].payload, &msg))
		require.Equal(t, "pong", msg["text"])
	}
}

// TestConnectBidiStream_HTTP1_BodyClosedAfterResponseStart documents the expected
// HTTP/1.1 limitation for bidi streaming: Go's HTTP/1.1 server closes r.Body
// once the handler begins writing the response, so subsequent reads of request
// frames fail with "invalid Read on closed Body". The Connect protocol spec
// mandates HTTP/2 for bidirectional streaming; this test pins that boundary.
func TestConnectBidiStream_HTTP1_BodyClosedAfterResponseStart(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "Chat",
		Input:   stuber.InputData{Equals: map[string]any{}},
		Output:  stuber.Output{Data: map[string]any{"user": "server", "text": "pong"}},
	}
	srv := startH2CServer(t, newChatConnectHandler(t, stub))

	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping"}),
		jsonEndStreamFrame(t),
	)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		srv.URL+"/chat.ChatService/Chat", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/connect+json")

	resp, err := http1Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, "HTTP/1.1", resp.Proto)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	frames := parseAllFrames(t, raw)
	// First response frame succeeds; the end-stream frame carries an internal
	// error because r.Body is already closed when the handler tries to read
	// the next request frame.
	require.Len(t, frames, 2)
	require.False(t, frames[0].isEndStream, "first frame must be a data frame")

	require.True(t, frames[1].isEndStream, "second frame must be end-stream")
	var errPayload map[string]any
	require.NoError(t, json.Unmarshal(frames[1].payload, &errPayload))
	errObj, ok := errPayload["error"].(map[string]any)
	require.True(t, ok, "end-stream payload must contain an error object")
	require.Equal(t, "internal", errObj["code"])
	require.Contains(t, errObj["message"].(string), "closed Body")
}

// ── Protocol negotiation ──────────────────────────────────────────────────────

// TestConnectProtocolNegotiation verifies that h2c correctly routes each client
// to the right HTTP version without any change to the underlying handler.
// It uses a minimal passthrough that records which protocol each request used,
// so we can assert that an HTTP/1.1 client really did stay on HTTP/1.1 and an
// h2c client really did upgrade to HTTP/2.
func TestConnectProtocolNegotiation(t *testing.T) {
	t.Parallel()

	protos := make(chan string, 2)

	probe := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protos <- r.Proto
		w.WriteHeader(http.StatusOK)
	})

	srv := startH2CServer(t, probe)

	doRequest := func(client *http.Client) {
		t.Helper()

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			srv.URL+"/probe", http.NoBody)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
	}

	doRequest(http1Client())
	doRequest(h2cClient())

	got := make([]string, 2)
	got[0] = <-protos
	got[1] = <-protos

	require.Contains(t, got, "HTTP/1.1", "HTTP/1.1 client should have been served HTTP/1.1")
	require.Contains(t, got, "HTTP/2.0", "h2c client should have been served HTTP/2.0")
}
