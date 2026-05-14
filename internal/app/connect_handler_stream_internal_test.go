package app

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// ── Frame helpers ─────────────────────────────────────────────────────────────

type parsedFrame struct {
	isEndStream bool
	payload     []byte
}

// parseAllFrames reads every Connect stream frame from body until EOF or the first end-stream frame.
func parseAllFrames(t *testing.T, body []byte) []parsedFrame {
	t.Helper()

	var frames []parsedFrame

	r := bytes.NewReader(body)

	for {
		payload, isEnd, err := readConnectStreamFrame(r)
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		frames = append(frames, parsedFrame{isEndStream: isEnd, payload: payload})

		if isEnd {
			break
		}
	}

	return frames
}

// jsonDataFrame encodes obj as JSON and wraps it in a Connect data frame (flag=0x00).
func jsonDataFrame(t *testing.T, obj map[string]any) []byte {
	t.Helper()

	data, err := json.Marshal(obj)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, writeConnectStreamFrame(&buf, 0x00, data))

	return buf.Bytes()
}

// jsonEndStreamFrame encodes an end-stream frame with the canonical "{}" payload.
func jsonEndStreamFrame(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, writeConnectStreamFrame(&buf, connectStreamFlagEndStream, []byte("{}")))

	return buf.Bytes()
}

// buildStreamBody concatenates pre-encoded Connect frames into a single io.Reader for use as a request body.
func buildStreamBody(frames ...[]byte) io.Reader {
	var buf bytes.Buffer

	for _, f := range frames {
		buf.Write(f)
	}

	return &buf
}

// gzipCompress compresses data with gzip and returns the compressed bytes.
func gzipCompress(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes()
}

// ── Registry helpers ──────────────────────────────────────────────────────────

// buildChatRegistry compiles the chat service proto and returns a descriptor registry
// containing chat.ChatService (SendMessage, ReceiveMessages, Chat).
func buildChatRegistry(t *testing.T) *descriptors.Registry {
	t.Helper()

	protoPath := filepath.Join("..", "..", "examples", "projects", "chat", "service.proto")
	fdss, err := protoset.Build(t.Context(), nil, []string{protoPath}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fdss)

	files, err := protodesc.NewFiles(fdss[0])
	require.NoError(t, err)

	reg := descriptors.NewRegistry()
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		reg.Register(fd)
		return true
	})

	return reg
}

// buildGreeterRegistry compiles the greeter proto and returns a descriptor registry.
func buildGreeterRegistry(t *testing.T) *descriptors.Registry {
	t.Helper()

	protoPath := filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto")
	fdss, err := protoset.Build(t.Context(), nil, []string{protoPath}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fdss)

	files, err := protodesc.NewFiles(fdss[0])
	require.NoError(t, err)

	reg := descriptors.NewRegistry()
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		reg.Register(fd)
		return true
	})

	return reg
}

// newChatConnectHandler returns a ConnectHandler wired to a budgerigar pre-loaded with stubs
// and the chat service descriptor registry.
func newChatConnectHandler(t *testing.T, stubs ...*stuber.Stub) *ConnectHandler {
	t.Helper()

	bud := stuber.NewBudgerigar()
	if len(stubs) > 0 {
		bud.PutMany(stubs...)
	}

	return NewConnectHandler(bud, buildChatRegistry(t), nil)
}

// doStreamRequest fires an HTTP POST at path with the given content-type and body against handler.
func doStreamRequest(t *testing.T, handler *ConnectHandler, path, ct string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://localhost"+path, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", ct)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// requireErrorEndStream asserts that frames contains exactly one end-stream frame whose payload
// carries a Connect error object with the given code string (e.g. "not_found").
func requireErrorEndStream(t *testing.T, frames []parsedFrame, wantCode string) {
	t.Helper()

	require.Len(t, frames, 1)
	require.True(t, frames[0].isEndStream, "expected an end-stream frame")

	var body map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &body))

	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "expected 'error' key in end-stream payload")
	require.Equal(t, wantCode, errObj["code"])
}

// ── parseConnectPath ──────────────────────────────────────────────────────────

func TestParseConnectPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		service string
		method  string
		ok      bool
	}{
		{"/pkg.Service/Method", "pkg.Service", "Method", true},
		{"/chat.ChatService/ReceiveMessages", "chat.ChatService", "ReceiveMessages", true},
		{"/a/b/c", "a/b", "c", true},
		{"/", "", "", false},
		{"", "", "", false},
		{"/only", "", "", false},
		{"/a/", "", "", false}, // empty method segment
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			svc, mth, ok := parseConnectPath(tt.path)
			require.Equal(t, tt.ok, ok)

			if tt.ok {
				require.Equal(t, tt.service, svc)
				require.Equal(t, tt.method, mth)
			}
		})
	}
}

// ── Frame read / write round-trips ────────────────────────────────────────────

func TestWriteReadStreamFrame_RoundTrip(t *testing.T) {
	t.Parallel()

	payload := []byte("hello connect world")

	var buf bytes.Buffer
	require.NoError(t, writeConnectStreamFrame(&buf, 0x00, payload))

	got, isEnd, err := readConnectStreamFrame(&buf)
	require.NoError(t, err)
	require.False(t, isEnd)
	require.Equal(t, payload, got)
}

func TestReadStreamFrame_EOF(t *testing.T) {
	t.Parallel()

	_, _, err := readConnectStreamFrame(bytes.NewReader(nil))
	require.ErrorIs(t, err, io.EOF)
}

func TestReadStreamFrame_EndStreamFlag(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeConnectStreamFrame(&buf, connectStreamFlagEndStream, []byte("{}")))

	_, isEnd, err := readConnectStreamFrame(&buf)
	require.NoError(t, err)
	require.True(t, isEnd)
}

func TestReadStreamFrame_GzipDecompressed(t *testing.T) {
	t.Parallel()

	original := []byte(`{"user":"alice"}`)
	compressed := gzipCompress(t, original)

	var buf bytes.Buffer
	require.NoError(t, writeConnectStreamFrame(&buf, connectStreamFlagCompressed, compressed))

	payload, isEnd, err := readConnectStreamFrame(&buf)
	require.NoError(t, err)
	require.False(t, isEnd)
	require.Equal(t, original, payload)
}

func TestWriteReadStreamFrame_EmptyPayload(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeConnectStreamFrame(&buf, 0x00, nil))

	got, isEnd, err := readConnectStreamFrame(&buf)
	require.NoError(t, err)
	require.False(t, isEnd)
	require.Empty(t, got)
}

// ── Server streaming ──────────────────────────────────────────────────────────

func TestHandleServerStream_Success(t *testing.T) {
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

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "alice"}))
	rec := doStreamRequest(t, handler, "/chat.ChatService/ReceiveMessages", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))

	frames := parseAllFrames(t, rec.Body.Bytes())
	require.Len(t, frames, 3, "expected 2 data frames + 1 end-stream")

	require.False(t, frames[0].isEndStream)
	require.False(t, frames[1].isEndStream)
	require.True(t, frames[2].isEndStream)

	var msg0 map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &msg0))
	require.Equal(t, "server", msg0["user"])
	require.Equal(t, "msg1", msg0["text"])

	var msg1 map[string]any
	require.NoError(t, json.Unmarshal(frames[1].payload, &msg1))
	require.Equal(t, "server", msg1["user"])
	require.Equal(t, "msg2", msg1["text"])

	require.Equal(t, []byte("{}"), frames[2].payload)
}

func TestHandleServerStream_SingleDataFallback(t *testing.T) {
	t.Parallel()

	// When output.stream is empty but output.data is set, the handler
	// uses data as a single stream item.
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "ReceiveMessages",
		Input:   stuber.InputData{Equals: map[string]any{"user": "bob"}},
		Output: stuber.Output{
			Data: map[string]any{"user": "server", "text": "hi bob"},
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "bob"}))
	rec := doStreamRequest(t, handler, "/chat.ChatService/ReceiveMessages", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	require.Len(t, frames, 2, "expected 1 data frame + 1 end-stream")

	require.False(t, frames[0].isEndStream)
	require.True(t, frames[1].isEndStream)

	var msg map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &msg))
	require.Equal(t, "hi bob", msg["text"])
}

func TestHandleServerStream_StubNotFound(t *testing.T) {
	t.Parallel()

	handler := newChatConnectHandler(t) // no stubs
	body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "nobody"}))
	rec := doStreamRequest(t, handler, "/chat.ChatService/ReceiveMessages", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "not_found")
}

func TestHandleServerStream_StubErrorField(t *testing.T) {
	t.Parallel()

	code := codes.PermissionDenied
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "ReceiveMessages",
		Input:   stuber.InputData{Equals: map[string]any{"user": "alice"}},
		Output: stuber.Output{
			Error: "access denied",
			Code:  &code,
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "alice"}))
	rec := doStreamRequest(t, handler, "/chat.ChatService/ReceiveMessages", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "permission_denied")
}

func TestHandleServerStream_NoDescriptor(t *testing.T) {
	t.Parallel()

	// Use an entirely unknown service so GlobalFiles can never satisfy the lookup
	// even when other parallel tests register real protos.
	handler := NewConnectHandler(stuber.NewBudgerigar(), descriptors.NewRegistry(), nil)

	body := buildStreamBody(jsonDataFrame(t, map[string]any{"field": "value"}))
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost,
		"http://localhost/test.NoSuchService/NoSuchMethod",
		body,
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/connect+json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "unimplemented")
}

func TestHandleServerStream_CustomResponseHeaders(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "ReceiveMessages",
		Input:   stuber.InputData{Equals: map[string]any{"user": "alice"}},
		Output: stuber.Output{
			Headers: map[string]string{"x-room": "lobby"},
			Stream:  []any{map[string]any{"user": "server", "text": "hi"}},
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "alice"}))
	rec := doStreamRequest(t, handler, "/chat.ChatService/ReceiveMessages", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "lobby", rec.Header().Get("x-room"))
}

// ── Client streaming ──────────────────────────────────────────────────────────

func TestHandleClientStream_Success(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "SendMessage",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"user": "alice", "text": "hello"}},
		},
		Output: stuber.Output{
			Data: map[string]any{"success": true, "message": "received"},
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "hello"}),
		jsonEndStreamFrame(t),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/SendMessage", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))

	frames := parseAllFrames(t, rec.Body.Bytes())
	require.Len(t, frames, 2, "expected 1 response frame + 1 end-stream")

	require.False(t, frames[0].isEndStream)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &resp))
	require.Equal(t, true, resp["success"])
	require.Equal(t, "received", resp["message"])

	require.True(t, frames[1].isEndStream)
	require.Equal(t, []byte("{}"), frames[1].payload)
}

func TestHandleClientStream_MultipleMessages(t *testing.T) {
	t.Parallel()

	// Both messages must arrive before the stub is evaluated.
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "SendMessage",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"user": "alice", "text": "msg1"}},
			{Equals: map[string]any{"user": "alice", "text": "msg2"}},
		},
		Output: stuber.Output{
			Data: map[string]any{"success": true, "message": "got both"},
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "msg1"}),
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "msg2"}),
		jsonEndStreamFrame(t),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/SendMessage", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	require.Len(t, frames, 2)
	require.False(t, frames[0].isEndStream)
	require.True(t, frames[1].isEndStream)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &resp))
	require.Equal(t, "got both", resp["message"])
}

func TestHandleClientStream_StubNotFound(t *testing.T) {
	t.Parallel()

	handler := newChatConnectHandler(t) // no stubs

	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "nobody", "text": "hi"}),
		jsonEndStreamFrame(t),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/SendMessage", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "not_found")
}

func TestHandleClientStream_StubErrorField(t *testing.T) {
	t.Parallel()

	code := codes.Internal
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "SendMessage",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"user": "alice", "text": "hello"}},
		},
		Output: stuber.Output{
			Error: "processing error",
			Code:  &code,
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "hello"}),
		jsonEndStreamFrame(t),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/SendMessage", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "internal")
}

func TestHandleClientStream_EOFWithoutEndStreamFrame(t *testing.T) {
	t.Parallel()

	// The handler must also terminate the loop when the body reader reaches EOF
	// (even without an explicit end-stream frame).
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "SendMessage",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"user": "alice", "text": "hello"}},
		},
		Output: stuber.Output{
			Data: map[string]any{"success": true, "message": "ok"},
		},
	}

	handler := newChatConnectHandler(t, stub)
	// No end-stream frame — body ends with just the data frame.
	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "hello"}),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/SendMessage", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	require.Len(t, frames, 2)
	require.False(t, frames[0].isEndStream)
	require.True(t, frames[1].isEndStream)
}

// ── Bidirectional streaming ───────────────────────────────────────────────────

func TestHandleBidiStream_Success(t *testing.T) {
	t.Parallel()

	// A stub with an empty Equals (nil matcher) matches any incoming message.
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "Chat",
		Input:   stuber.InputData{Equals: map[string]any{}},
		Output: stuber.Output{
			Data: map[string]any{"user": "server", "text": "pong"},
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping"}),
		jsonEndStreamFrame(t),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/Chat", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))

	frames := parseAllFrames(t, rec.Body.Bytes())
	// 1 response frame + 1 end-stream
	require.Len(t, frames, 2)
	require.False(t, frames[0].isEndStream)

	var msg map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &msg))
	require.Equal(t, "server", msg["user"])
	require.Equal(t, "pong", msg["text"])

	require.True(t, frames[1].isEndStream)
	require.Equal(t, []byte("{}"), frames[1].payload)
}

func TestHandleBidiStream_MultipleRequestsProduceMultipleResponses(t *testing.T) {
	t.Parallel()

	// Each request message receives an independent response because the stub resets
	// its match state after every non-client-stream match.
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "Chat",
		Input:   stuber.InputData{Equals: map[string]any{}},
		Output: stuber.Output{
			Data: map[string]any{"user": "server", "text": "pong"},
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping1"}),
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping2"}),
		jsonEndStreamFrame(t),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/Chat", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	// 2 response frames + 1 end-stream
	require.Len(t, frames, 3)
	require.False(t, frames[0].isEndStream)
	require.False(t, frames[1].isEndStream)
	require.True(t, frames[2].isEndStream)
}

func TestHandleBidiStream_NoStubFound(t *testing.T) {
	t.Parallel()

	// No stubs registered → FindByQueryBidi returns an error → error end-stream.
	handler := newChatConnectHandler(t)
	body := buildStreamBody(
		jsonDataFrame(t, map[string]any{"user": "alice", "text": "ping"}),
		jsonEndStreamFrame(t),
	)
	rec := doStreamRequest(t, handler, "/chat.ChatService/Chat", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "not_found")
}

func TestHandleBidiStream_EmptyBodyProducesEndStream(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "Chat",
		Input:   stuber.InputData{Equals: map[string]any{}},
		Output: stuber.Output{
			Data: map[string]any{"user": "server", "text": "pong"},
		},
	}

	handler := newChatConnectHandler(t, stub)
	// Send only the end-stream frame (no data messages).
	body := buildStreamBody(jsonEndStreamFrame(t))
	rec := doStreamRequest(t, handler, "/chat.ChatService/Chat", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	// No messages processed → only end-stream frame.
	require.Len(t, frames, 1)
	require.True(t, frames[0].isEndStream)
	require.Equal(t, []byte("{}"), frames[0].payload)
}

// ── Routing via handleConnectStream ──────────────────────────────────────────

func TestHandleConnectStream_NoDescriptor_ReturnsUnimplemented(t *testing.T) {
	t.Parallel()

	// Use an entirely unknown service so GlobalFiles can never satisfy the lookup
	// even when other parallel tests register real protos.
	handler := NewConnectHandler(stuber.NewBudgerigar(), descriptors.NewRegistry(), nil)

	body := buildStreamBody(jsonDataFrame(t, map[string]any{"field": "value"}))
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost,
		"http://localhost/test.NoSuchService/NoSuchMethod",
		body,
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/connect+proto")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "unimplemented")
}

func TestHandleConnectStream_UnaryMethodCalledAsStream_ReturnsInvalidArgument(t *testing.T) {
	t.Parallel()

	// helloworld.Greeter/SayHello is a unary method. Calling it with application/connect+json
	// routes to handleConnectStream → dispatcher → default branch → invalid_argument.
	handler := NewConnectHandler(stuber.NewBudgerigar(), buildGreeterRegistry(t), nil)

	body := buildStreamBody(jsonDataFrame(t, map[string]any{"name": "alice"}))
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost,
		"http://localhost/helloworld.Greeter/SayHello",
		body,
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/connect+json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	frames := parseAllFrames(t, rec.Body.Bytes())
	requireErrorEndStream(t, frames, "invalid_argument")
}

func TestHandleConnectStream_JSONEncoding(t *testing.T) {
	t.Parallel()

	// End-to-end test verifying that application/connect+json uses JSON frames for both
	// request and response in a server-streaming scenario.
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "chat.ChatService",
		Method:  "ReceiveMessages",
		Input:   stuber.InputData{Equals: map[string]any{"user": "json-client"}},
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"user": "srv", "text": "hello from json"},
			},
		},
	}

	handler := newChatConnectHandler(t, stub)
	body := buildStreamBody(jsonDataFrame(t, map[string]any{"user": "json-client"}))
	rec := doStreamRequest(t, handler, "/chat.ChatService/ReceiveMessages", "application/connect+json", body)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/connect+json", rec.Header().Get("Content-Type"))

	frames := parseAllFrames(t, rec.Body.Bytes())
	require.Len(t, frames, 2)

	// Verify the data frame carries a parseable JSON object.
	var msg map[string]any
	require.NoError(t, json.Unmarshal(frames[0].payload, &msg))
	require.Equal(t, "hello from json", msg["text"])
}
