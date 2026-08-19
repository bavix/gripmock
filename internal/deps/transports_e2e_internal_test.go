package deps

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/config"
	protodom "github.com/bavix/gripmock/v3/internal/domain/proto"
	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const e2eProto = `
syntax = "proto3";
package e2e;
import "google/protobuf/field_mask.proto";
service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply);
  rpc Peek (HelloRequest) returns (HelloReply) { option idempotency_level = NO_SIDE_EFFECTS; }
  rpc Watch (WatchRequest) returns (stream WatchEvent);
  rpc Collect (stream Chunk) returns (Summary);
  rpc Chat (stream ChatMessage) returns (stream ChatMessage);
}
message HelloRequest { string name = 1; int64 count = 2; google.protobuf.FieldMask mask = 3; }
message HelloReply { string message = 1; }
message WatchRequest { string topic = 1; }
message WatchEvent { string payload = 1; }
message Chunk { string part = 1; }
message Summary { string digest = 1; }
message ChatMessage { string text = 1; }
`

const shutdownBudget = 2 * time.Second

type e2eServer struct {
	grpcAddr    string
	restURL     string
	gatewayAddr string
	protoPath   string
}

func freeAddr(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	return addr
}

func startAllTransports(t *testing.T) *e2eServer {
	t.Helper()

	dir := t.TempDir()
	protoPath := filepath.Join(dir, "e2e.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(e2eProto), 0o600))

	cfg := config.Load()
	cfg.GRPC.Addr = freeAddr(t)
	cfg.HTTP.Addr = freeAddr(t)
	cfg.Gateway.Addr = freeAddr(t)

	builder := NewBuilder(WithConfig(cfg))

	ctx, cancel := context.WithCancel(t.Context())

	errs := make(chan error, 3)

	go func() {
		rest, err := builder.RestServe(ctx, "")
		if err != nil {
			errs <- err

			return
		}

		errs <- rest.ListenAndServe()
	}()
	go func() { errs <- builder.GatewayServe(ctx) }()
	go func() { errs <- builder.GRPCServe(ctx, protodom.New([]string{protoPath}, nil, nil)) }()

	go func() {
		for err := range errs {
			if err != nil && ctx.Err() == nil {
				t.Log("transport server stopped: ", err)
			}
		}
	}()

	t.Cleanup(func() {
		cancel()

		stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(context.Background()), shutdownBudget)
		defer stopCancel()

		builder.Shutdown(stopCtx)
	})

	srv := &e2eServer{
		grpcAddr:    cfg.GRPC.Addr,
		restURL:     "http://" + cfg.HTTP.Addr,
		gatewayAddr: cfg.Gateway.Addr,
		protoPath:   protoPath,
	}

	waitReady(t, srv)

	return srv
}

func waitReady(t *testing.T, srv *e2eServer) {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	dialer := net.Dialer{Timeout: 100 * time.Millisecond}

	for name, addr := range map[string]string{
		"grpc":    srv.grpcAddr,
		"gateway": srv.gatewayAddr,
		"rest":    strings.TrimPrefix(srv.restURL, "http://"),
	} {
		for {
			conn, err := dialer.DialContext(t.Context(), "tcp", addr)
			if err == nil {
				_ = conn.Close()

				break
			}

			if time.Now().After(deadline) {
				t.Fatalf("%s (%s) did not come up: %v", name, addr, err)
			}

			time.Sleep(20 * time.Millisecond)
		}
	}

	for {
		if serviceKnown(t, srv) {
			return
		}

		if time.Now().After(deadline) {
			t.Fatal("e2e.Greeter never appeared in the service list")
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func serviceKnown(t *testing.T, srv *e2eServer) bool {
	t.Helper()

	var services []struct {
		ID string `json:"id"`
	}

	if !getJSON(t, srv.restURL+"/api/services", &services) {
		return false
	}

	for _, service := range services {
		if service.ID == "e2e.Greeter" {
			return true
		}
	}

	return false
}

func getJSON(t *testing.T, url string, out any) bool {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	require.NoError(t, json.NewDecoder(resp.Body).Decode(out))

	return true
}

func postJSON(t *testing.T, url string, body any, out any, headers ...[2]string) int {
	t.Helper()

	encoded, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(encoded))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	for _, header := range headers {
		req.Header.Set(header[0], header[1])
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	if resp.StatusCode >= http.StatusMultipleChoices {
		t.Log("response ", resp.StatusCode, ": ", string(raw))
	} else if out != nil {
		require.NoError(t, json.Unmarshal(raw, out))
	}

	return resp.StatusCode
}

func (s *e2eServer) addStub(t *testing.T, name, message string) {
	t.Helper()

	s.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": name}},
		"output":  map[string]any{"data": map[string]any{"message": message}},
	}, "")
}

func (s *e2eServer) putStub(t *testing.T, stub map[string]any, session string) {
	t.Helper()

	headers := [][2]string{}
	if session != "" {
		headers = append(headers, [2]string{"X-Gripmock-Session", session})
	}

	code := postJSON(t, s.restURL+"/api/stubs", []map[string]any{stub}, nil, headers...)
	require.Equal(t, http.StatusOK, code)
}

//nolint:ireturn // protoreflect returns interfaces
func messagePair(t *testing.T, path, inName, outName string) (protoreflect.MessageDescriptor, protoreflect.MessageDescriptor) {
	t.Helper()

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{filepath.Dir(path)},
		}),
	}

	files, err := compiler.Compile(t.Context(), filepath.Base(path))
	require.NoError(t, err)
	require.Len(t, files, 1)

	messages := files[0].Messages()

	return messages.ByName(protoreflect.Name(inName)), messages.ByName(protoreflect.Name(outName))
}

//nolint:ireturn // protoreflect returns interfaces
func compileE2EDescriptors(t *testing.T, path string) (protoreflect.MessageDescriptor, protoreflect.MessageDescriptor) {
	t.Helper()

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{filepath.Dir(path)},
		}),
	}

	files, err := compiler.Compile(t.Context(), filepath.Base(path))
	require.NoError(t, err)
	require.Len(t, files, 1)

	messages := files[0].Messages()

	return messages.ByName("HelloRequest"), messages.ByName("HelloReply")
}

func buildWebFrame(payload []byte) []byte {
	frame := make([]byte, 5+len(payload))
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload))) //nolint:gosec // test payloads are tiny
	copy(frame[5:], payload)

	return frame
}

func TestAllTransportsServeTheSameStub(t *testing.T) { //nolint:paralleltest // one shared server, ordered legs
	srv := startAllTransports(t)
	srv.addStub(t, "Alex", "Hello Alex")

	in, out := compileE2EDescriptors(t, srv.protoPath)

	//nolint:paralleltest // the legs share one server and must run in order
	for _, leg := range []struct {
		name string
		run  func(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor)
	}{
		{"grpc", checkGRPC},
		{"connectrpc", checkConnect},
		{"connectrpc error", checkConnectError},
		{"grpc-web", checkGRPCWeb},
		{"rest history", checkRestHistory},
		{"rest search", checkRestSearch},
		{"embedded sdk", checkEmbeddedSDK},
		{"gateway metadata", checkGatewayMetadata},
		{"gateway session", checkGatewaySession},
		{"connect error details", checkConnectErrorDetails},
		{"gateway server stream", checkGatewayServerStream},
		{"grpc-web text", checkGRPCWebText},
		{"connect get", checkConnectGet},
		{"grpc error details", checkGRPCErrorDetails},
		{"mcp", checkMCP},
		{"mcp stub workflow", checkMCPStubWorkflow},
		{"rest stub lifecycle", checkRestStubLifecycle},
		{"gateway templates and headers", checkGatewayTemplatesAndHeaders},
		{"gateway call budget and effects", checkGatewayCallBudgetAndEffects},
		{"gateway delay", checkGatewayDelay},
		{"grpc-web gzip frame", checkGRPCWebGzipFrame},
		{"rest error filter and ui", checkRestErrorFilterAndUI},
		{"grpc reflection and health", checkGRPCReflectionAndHealth},
		{"dynamic descriptor upload", checkDynamicDescriptorUpload},
		{"sessions list", checkSessionsList},
		{"grpc client stream", checkGRPCClientStream},
		{"grpc bidi stream", checkGRPCBidiStream},
	} {
		t.Run(leg.name, func(t *testing.T) {
			leg.run(t, srv, in, out)
		})
	}
}

func checkGRPC(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	conn, err := grpc.NewClient(srv.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

	reply := dynamicpb.NewMessage(out)

	require.NoError(t, conn.Invoke(t.Context(), "/e2e.Greeter/SayHello", request, reply))
	require.Equal(t, "Hello Alex", reply.Get(out.Fields().ByName("message")).String())

	unmatched := dynamicpb.NewMessage(in)
	unmatched.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Nobody"))
	err = conn.Invoke(t.Context(), "/e2e.Greeter/SayHello", unmatched, dynamicpb.NewMessage(out))
	require.Equal(t, codes.NotFound, status.Code(err))
}

func checkConnect(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	resp := connectJSON(t, srv, "SayHello", `{"name":"Alex"}`)
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))
	require.JSONEq(t, `{"message":"Hello Alex"}`, string(resp.body))
}

func checkConnectError(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	resp := connectJSON(t, srv, "SayHello", `{"name":"Nobody"}`)
	require.Equal(t, http.StatusNotFound, resp.status)
	require.Contains(t, string(resp.body), `"code":"not_found"`)
}

func checkGRPCWeb(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

	payload, err := proto.Marshal(request)
	require.NoError(t, err)

	resp := gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/SayHello",
		"application/grpc-web+proto", string(buildWebFrame(payload)))
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))

	frames := readFrames(t, resp.body)
	require.Len(t, frames, 2)
	require.Equal(t, byte(0x00), frames[0][0], "first frame must carry the message")

	reply := dynamicpb.NewMessage(out)
	require.NoError(t, proto.Unmarshal(frameBody(t, frames[0]), reply))
	require.Equal(t, "Hello Alex", reply.Get(out.Fields().ByName("message")).String())

	require.Equal(t, byte(0x80), frames[1][0], "second frame must carry the trailers")
	require.Contains(t, string(frameBody(t, frames[1])), "grpc-status: 0")
}

func checkRestHistory(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	var records []map[string]any

	require.True(t, getJSON(t, srv.restURL+"/api/history", &records))
	require.NotEmpty(t, records)

	var served, unmatched int

	for _, record := range records {
		if method, _ := record["method"].(string); method != "SayHello" {
			continue
		}

		if code, _ := record["code"].(float64); code == float64(codes.NotFound) {
			unmatched++

			continue
		}

		served++
	}

	require.GreaterOrEqual(t, served, 3, "grpc, connect and grpc-web must be recorded")
	require.GreaterOrEqual(t, unmatched, 2, "calls that matched no stub must be recorded too")
}

func checkRestSearch(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	var found map[string]any

	code := postJSON(t, srv.restURL+"/api/stubs/search", map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"data":    map[string]any{"name": "Alex"},
	}, &found)
	require.Equal(t, http.StatusOK, code)

	data, ok := found["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Hello Alex", data["message"])
}

func checkEmbeddedSDK(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	embedded := sdk.NewServer(t, sdk.WithProtoFiles(srv.protoPath))

	embedded.ExpectUnary("/e2e.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hello Alex")

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

	reply := dynamicpb.NewMessage(out)

	require.NoError(t, embedded.Conn().Invoke(t.Context(), "/e2e.Greeter/SayHello", request, reply))
	require.Equal(t, "Hello Alex", reply.Get(out.Fields().ByName("message")).String())
	require.Equal(t, 1, embedded.Called("/e2e.Greeter/SayHello"))
}

func checkMCP(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	result := srv.mcpCall(t, "history_list", map[string]any{"service": "e2e.Greeter"})

	records, ok := result["records"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, records)

	mocked := srv.mcpCall(t, "mock_call", map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"payload": map[string]any{"name": "Alex"},
	})
	require.Equal(t, true, mocked["matched"])

	data, ok := mocked["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Hello Alex", data["message"])
}

func (s *e2eServer) mcpCall(t *testing.T, tool string, args map[string]any) map[string]any {
	t.Helper()

	var envelope struct {
		Result struct {
			StructuredContent map[string]any `json:"structuredContent"`
			IsError           bool           `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	code := postJSON(t, s.restURL+"/api/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": tool, "arguments": args},
	}, &envelope, [2]string{"Accept", "application/json, text/event-stream"})
	require.Equal(t, http.StatusOK, code)
	require.Nilf(t, envelope.Error, "tool %s failed", tool)
	require.Falsef(t, envelope.Result.IsError, "tool %s returned an error result", tool)

	return envelope.Result.StructuredContent
}

type httpResult struct {
	status int
	header http.Header
	body   []byte
}

func gatewayPost(t *testing.T, url, contentType, body string, headers ...[2]string) httpResult {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)

	for _, header := range headers {
		req.Header.Set(header[0], header[1])
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return httpResult{status: resp.StatusCode, header: resp.Header, body: raw}
}

func connectJSON(t *testing.T, srv *e2eServer, method, body string, headers ...[2]string) httpResult {
	t.Helper()

	all := append([][2]string{{"Connect-Protocol-Version", "1"}}, headers...)

	return gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/"+method, "application/json", body, all...)
}

func readFrames(t *testing.T, body []byte) [][2]any {
	t.Helper()

	frames := make([][2]any, 0, 2)

	for len(body) >= 5 {
		size := binary.BigEndian.Uint32(body[1:5])
		require.LessOrEqual(t, int(size)+5, len(body), "frame runs past the body")

		frames = append(frames, [2]any{body[0], body[5 : 5+size]})
		body = body[5+size:]
	}

	return frames
}

func checkGatewayMetadata(t *testing.T, srv *e2eServer, in, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Meta"}},
		"output": map[string]any{
			"data":     map[string]any{"message": "Hello Meta"},
			"headers":  map[string]any{"x-shard": "eu-3"},
			"trailers": map[string]any{"x-cost": "2"},
		},
	}, "")

	resp := connectJSON(t, srv, "SayHello", `{"name":"Meta"}`)
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))
	require.Equal(t, "eu-3", resp.header.Get("X-Shard"))
	require.Equal(t, "2", resp.header.Get("Trailer-X-Cost"))

	request := dynamicpb.NewMessage(in)
	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Meta"))

	payload, err := proto.Marshal(request)
	require.NoError(t, err)

	webResp := gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/SayHello",
		"application/grpc-web+proto", string(buildWebFrame(payload)))
	require.Equal(t, http.StatusOK, webResp.status)
	require.Equal(t, "eu-3", webResp.header.Get("X-Shard"))

	frames := readFrames(t, webResp.body)
	require.Len(t, frames, 2)

	require.Contains(t, string(frameBody(t, frames[1])), "x-cost: 2")
}

func checkGatewaySession(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Scoped"}},
		"output":  map[string]any{"data": map[string]any{"message": "scoped reply"}},
	}, "gateway-session")

	resp := connectJSON(t, srv, "SayHello", `{"name":"Scoped"}`,
		[2]string{"X-Gripmock-Session", "gateway-session"})
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))
	require.JSONEq(t, `{"message":"scoped reply"}`, string(resp.body))

	blind := connectJSON(t, srv, "SayHello", `{"name":"Scoped"}`)
	require.Equal(t, http.StatusNotFound, blind.status, "a session stub must stay invisible globally")

	other := connectJSON(t, srv, "SayHello", `{"name":"Scoped"}`,
		[2]string{"X-Gripmock-Session", "another-session"})
	require.Equal(t, http.StatusNotFound, other.status)
}

func checkConnectErrorDetails(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Broke"}},
		"output": map[string]any{
			"error": "quota exhausted",
			"code":  8,
			"details": []any{map[string]any{
				"type":   "type.googleapis.com/google.rpc.ErrorInfo",
				"reason": "QUOTA",
				"domain": "e2e",
			}},
		},
	}, "")

	resp := connectJSON(t, srv, "SayHello", `{"name":"Broke"}`)
	require.Equal(t, http.StatusTooManyRequests, resp.status, string(resp.body))

	// The protocol carries a detail as its type name plus the serialized message,
	// with the protobuf JSON rendering relegated to the optional debug field.
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Type  string          `json:"type"`
			Value string          `json:"value"`
			Debug json.RawMessage `json:"debug"`
		} `json:"details"`
	}

	require.NoError(t, json.Unmarshal(resp.body, &body))
	require.Equal(t, "resource_exhausted", body.Code)
	require.Equal(t, "quota exhausted", body.Message)
	require.Len(t, body.Details, 1)
	require.Equal(t, "google.rpc.ErrorInfo", body.Details[0].Type)
	require.JSONEq(t, `{
		"@type": "type.googleapis.com/google.rpc.ErrorInfo",
		"domain": "e2e",
		"reason": "QUOTA"
	}`, string(body.Details[0].Debug))

	// The value must be the detail itself. Asserting the base64 literally would
	// pin a field order protobuf does not promise, so decode it back instead.
	raw, err := base64.RawStdEncoding.DecodeString(body.Details[0].Value)
	require.NoError(t, err)

	var info errdetails.ErrorInfo

	require.NoError(t, proto.Unmarshal(raw, &info))
	require.Equal(t, "QUOTA", info.GetReason())
	require.Equal(t, "e2e", info.GetDomain())
}

func checkGatewayServerStream(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "Watch",
		"input":   map[string]any{"equals": map[string]any{"topic": "orders"}},
		"output": map[string]any{"stream": []any{
			map[string]any{"payload": "first"},
			map[string]any{"payload": "second"},
		}},
	}, "")

	resp := gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/Watch",
		"application/connect+json", string(buildWebFrame([]byte(`{"topic":"orders"}`))),
		[2]string{"Connect-Protocol-Version", "1"})
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))

	frames := readFrames(t, resp.body)
	require.Len(t, frames, 3, "two messages and the end-of-stream envelope")
	require.Contains(t, string(frameBody(t, frames[0])), "first")
	require.Contains(t, string(frameBody(t, frames[1])), "second")
	require.Equal(t, byte(0x02), frames[2][0], "last envelope must be end-of-stream")
}

func checkGRPCWebText(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	request := dynamicpb.NewMessage(in)
	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

	payload, err := proto.Marshal(request)
	require.NoError(t, err)

	encoded := base64.StdEncoding.EncodeToString(buildWebFrame(payload))

	resp := gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/SayHello",
		"application/grpc-web-text+proto", encoded)
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))

	decoded, err := base64.StdEncoding.DecodeString(string(resp.body))
	require.NoError(t, err)

	frames := readFrames(t, decoded)
	require.GreaterOrEqual(t, len(frames), 2)

	reply := dynamicpb.NewMessage(out)
	require.NoError(t, proto.Unmarshal(frameBody(t, frames[0]), reply))
	require.Equal(t, "Hello Alex", reply.Get(out.Fields().ByName("message")).String())
}

func checkConnectGet(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "Peek",
		"input":   map[string]any{"equals": map[string]any{"name": "Alex"}},
		"output":  map[string]any{"data": map[string]any{"message": "peeked"}},
	}, "")

	query := url.Values{}
	query.Set("encoding", "json")
	query.Set("message", `{"name":"Alex"}`)
	query.Set("connect", "v1")

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		"http://"+srv.gatewayAddr+"/e2e.Greeter/Peek?"+query.Encode(), nil)
	require.NoError(t, err)

	raw, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = raw.Body.Close() }()

	body, err := io.ReadAll(raw.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, raw.StatusCode, string(body))
	require.JSONEq(t, `{"message":"peeked"}`, string(body))

	post := connectJSON(t, srv, "Peek", `{"name":"Alex"}`)
	require.Equal(t, http.StatusOK, post.status, string(post.body))
	require.JSONEq(t, `{"message":"peeked"}`, string(post.body), "GET and POST must agree")
}

func checkGRPCErrorDetails(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	conn, err := grpc.NewClient(srv.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Broke"))

	err = conn.Invoke(t.Context(), "/e2e.Greeter/SayHello", request, dynamicpb.NewMessage(out))
	require.Error(t, err)

	st := status.Convert(err)
	require.Equal(t, codes.ResourceExhausted, st.Code())
	require.Equal(t, "quota exhausted", st.Message())
	require.Len(t, st.Details(), 1)

	info, ok := st.Details()[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "QUOTA", info.GetReason())
	require.Equal(t, "e2e", info.GetDomain())
}

func checkMCPStubWorkflow(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	created := srv.mcpCall(t, "stubs_upsert", map[string]any{
		"stubs": map[string]any{
			"service": "e2e.Greeter",
			"method":  "SayHello",
			"input":   map[string]any{"equals": map[string]any{"name": "FromMCP"}},
			"output":  map[string]any{"data": map[string]any{"message": "made by an agent"}},
		},
	})
	ids, ok := created["ids"].([]any)
	require.True(t, ok)
	require.Len(t, ids, 1)

	conn, err := grpc.NewClient(srv.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("FromMCP"))

	reply := dynamicpb.NewMessage(out)
	require.NoError(t, conn.Invoke(t.Context(), "/e2e.Greeter/SayHello", request, reply))
	require.Equal(t, "made by an agent", reply.Get(out.Fields().ByName("message")).String())

	verified := srv.mcpCall(t, "verify_calls", map[string]any{
		"service":       "e2e.Greeter",
		"method":        "SayHello",
		"expectedCount": 1,
	})
	require.Equal(t, false, verified["verified"], "the method was called more than once by now")

	inspected := srv.mcpCall(t, "stubs_inspect", map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   []any{map[string]any{"name": "FromMCP"}},
	})
	require.NotEmpty(t, inspected)

	deleted := srv.mcpCall(t, "stubs_delete", map[string]any{"id": ids[0]})
	require.Equal(t, true, deleted["deleted"])
}

func checkRestStubLifecycle(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	var invalid map[string]any

	code := postJSON(t, srv.restURL+"/api/stubs/validate", []map[string]any{{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Valid"}},
		"inputs":  []any{map[string]any{"equals": map[string]any{"name": "Valid"}}},
		"output":  map[string]any{"data": map[string]any{"message": "ok"}},
	}}, &invalid)
	require.Equal(t, http.StatusBadRequest, code, "input and inputs together must be rejected")

	var used, unused []map[string]any

	require.True(t, getJSON(t, srv.restURL+"/api/stubs/used", &used))
	require.True(t, getJSON(t, srv.restURL+"/api/stubs/unused", &unused))
	require.NotEmpty(t, used, "the calls above must have marked stubs as used")

	var report map[string]any

	code = postJSON(t, srv.restURL+"/api/stubs/inspect", map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"data":    map[string]any{"name": "NoSuchName"},
	}, &report)
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, report)
}

func checkGatewayTemplatesAndHeaders(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"headers": map[string]any{"equals": map[string]any{"x-tenant": "acme"}},
		"input":   map[string]any{"equals": map[string]any{"name": "Tenant"}},
		"output":  map[string]any{"data": map[string]any{"message": "Hello {{.Request.name}}"}},
	}, "")

	resp := connectJSON(t, srv, "SayHello", `{"name":"Tenant"}`, [2]string{"X-Tenant", "acme"})
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))
	require.JSONEq(t, `{"message":"Hello Tenant"}`, string(resp.body))

	blind := connectJSON(t, srv, "SayHello", `{"name":"Tenant"}`)
	require.Equal(t, http.StatusNotFound, blind.status, "a header-gated stub must not answer without it")
}

func checkGatewayCallBudgetAndEffects(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Once"}},
		"output":  map[string]any{"data": map[string]any{"message": "first and last"}},
		"options": map[string]any{"times": 1},
		"effects": []any{map[string]any{
			"action": "upsert",
			"stub": map[string]any{
				"service": "e2e.Greeter",
				"method":  "SayHello",
				"input":   map[string]any{"equals": map[string]any{"name": "Unlocked"}},
				"output":  map[string]any{"data": map[string]any{"message": "unlocked"}},
			},
		}},
	}, "")

	locked := connectJSON(t, srv, "SayHello", `{"name":"Unlocked"}`)
	require.Equal(t, http.StatusNotFound, locked.status)

	resp := connectJSON(t, srv, "SayHello", `{"name":"Once"}`)
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))

	spent := connectJSON(t, srv, "SayHello", `{"name":"Once"}`)
	require.Equal(t, http.StatusNotFound, spent.status, "times:1 must be spent")

	unlocked := connectJSON(t, srv, "SayHello", `{"name":"Unlocked"}`)
	require.Equal(t, http.StatusOK, unlocked.status, string(unlocked.body))
	require.JSONEq(t, `{"message":"unlocked"}`, string(unlocked.body))
}

func checkGatewayDelay(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	const pause = 120 * time.Millisecond

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Slow"}},
		"output":  map[string]any{"data": map[string]any{"message": "late"}, "delay": pause.String()},
	}, "")

	started := time.Now()

	resp := connectJSON(t, srv, "SayHello", `{"name":"Slow"}`)
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))
	require.GreaterOrEqual(t, time.Since(started), pause)
}

func checkGRPCWebGzipFrame(t *testing.T, srv *e2eServer, in, out protoreflect.MessageDescriptor) {
	t.Helper()

	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

	payload, err := proto.Marshal(request)
	require.NoError(t, err)

	var compressed bytes.Buffer

	writer := gzip.NewWriter(&compressed)
	_, err = writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	frame := buildWebFrame(compressed.Bytes())
	frame[0] = 0x01

	resp := gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/SayHello",
		"application/grpc-web+proto", string(frame), [2]string{"grpc-encoding", "gzip"})
	require.Equal(t, http.StatusOK, resp.status, string(resp.body))

	frames := readFrames(t, resp.body)
	require.GreaterOrEqual(t, len(frames), 1)

	reply := dynamicpb.NewMessage(out)
	require.NoError(t, proto.Unmarshal(frameBody(t, frames[0]), reply))
	require.Equal(t, "Hello Alex", reply.Get(out.Fields().ByName("message")).String())
}

func checkRestErrorFilterAndUI(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	var failures []map[string]any

	require.True(t, getJSON(t, srv.restURL+"/api/history?error=true", &failures))
	require.NotEmpty(t, failures, "the unmatched calls must be reachable through the error filter")

	for _, record := range failures {
		code, _ := record["code"].(float64)
		require.NotEqual(t, float64(codes.OK), code)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.restURL+"/", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "the UI must be served from the same port")
}

func checkGRPCReflectionAndHealth(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	conn, err := grpc.NewClient(srv.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	health, err := healthpb.NewHealthClient(conn).Check(t.Context(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, health.GetStatus())

	stream, err := reflectionpb.NewServerReflectionClient(conn).ServerReflectionInfo(t.Context())
	require.NoError(t, err)
	require.NoError(t, stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{},
	}))

	resp, err := stream.Recv()
	require.NoError(t, err)

	listed := resp.GetListServicesResponse().GetService()

	services := make([]string, 0, len(listed))
	for _, service := range listed {
		services = append(services, service.GetName())
	}

	require.Contains(t, services, "e2e.Greeter")
}

func writeLateProto(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "late.proto")
	require.NoError(t, os.WriteFile(path, []byte(`
syntax = "proto3";
package late;
service Late {
  rpc Ping (PingRequest) returns (PingReply);
}
message PingRequest { string id = 1; }
message PingReply { string pong = 1; }
`), 0o600))

	return path
}

func checkDynamicDescriptorUpload(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	path := writeLateProto(t)

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{filepath.Dir(path)},
		}),
	}

	files, err := compiler.Compile(t.Context(), filepath.Base(path))
	require.NoError(t, err)

	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{protodesc.ToFileDescriptorProto(files[0])}}

	encoded, err := proto.Marshal(set)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		srv.restURL+"/api/descriptors", bytes.NewReader(encoded))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	srv.putStub(t, map[string]any{
		"service": "late.Late",
		"method":  "Ping",
		"input":   map[string]any{"equals": map[string]any{"id": "1"}},
		"output":  map[string]any{"data": map[string]any{"pong": "late pong"}},
	}, "")

	conn, err := grpc.NewClient(srv.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	messages := files[0].Messages()
	inDesc := messages.ByName("PingRequest")
	outDesc := messages.ByName("PingReply")

	request := dynamicpb.NewMessage(inDesc)

	request.Set(inDesc.Fields().ByName("id"), protoreflect.ValueOfString("1"))

	reply := dynamicpb.NewMessage(outDesc)
	require.NoError(t, conn.Invoke(t.Context(), "/late.Late/Ping", request, reply))
	require.Equal(t, "late pong", reply.Get(outDesc.Fields().ByName("pong")).String())
}

func checkSessionsList(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	var payload struct {
		Sessions []string `json:"sessions"`
	}

	require.True(t, getJSON(t, srv.restURL+"/api/sessions", &payload))
	require.Contains(t, payload.Sessions, "gateway-session")
}

func (s *e2eServer) dial(t *testing.T) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.NewClient(s.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func checkGRPCClientStream(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "Collect",
		"inputs": []any{
			map[string]any{"equals": map[string]any{"part": "a"}},
			map[string]any{"equals": map[string]any{"part": "b"}},
		},
		"output": map[string]any{"data": map[string]any{"digest": "ab"}},
	}, "")

	chunk, summary := messagePair(t, srv.protoPath, "Chunk", "Summary")
	conn := srv.dial(t)

	stream, err := conn.NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Collect", ClientStreams: true}, "/e2e.Greeter/Collect")
	require.NoError(t, err)

	for _, part := range []string{"a", "b"} {
		msg := dynamicpb.NewMessage(chunk)

		msg.Set(chunk.Fields().ByName("part"), protoreflect.ValueOfString(part))
		require.NoError(t, stream.SendMsg(msg))
	}

	require.NoError(t, stream.CloseSend())

	reply := dynamicpb.NewMessage(summary)
	require.NoError(t, stream.RecvMsg(reply))
	require.Equal(t, "ab", reply.Get(summary.Fields().ByName("digest")).String())

	reversed, err := conn.NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Collect", ClientStreams: true}, "/e2e.Greeter/Collect")
	require.NoError(t, err)

	for _, part := range []string{"b", "a"} {
		msg := dynamicpb.NewMessage(chunk)

		msg.Set(chunk.Fields().ByName("part"), protoreflect.ValueOfString(part))
		require.NoError(t, reversed.SendMsg(msg))
	}

	require.NoError(t, reversed.CloseSend())
	require.Equal(t, codes.NotFound, status.Code(reversed.RecvMsg(dynamicpb.NewMessage(summary))))
}

func checkGRPCBidiStream(t *testing.T, srv *e2eServer, _, _ protoreflect.MessageDescriptor) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "Chat",
		"inputs":  []any{map[string]any{"equals": map[string]any{"text": "ping"}}},
		"output":  map[string]any{"stream": []any{map[string]any{"text": "pong"}}},
	}, "")

	chat, _ := messagePair(t, srv.protoPath, "ChatMessage", "ChatMessage")
	conn := srv.dial(t)

	stream, err := conn.NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true}, "/e2e.Greeter/Chat")
	require.NoError(t, err)

	msg := dynamicpb.NewMessage(chat)

	msg.Set(chat.Fields().ByName("text"), protoreflect.ValueOfString("ping"))
	require.NoError(t, stream.SendMsg(msg))
	require.NoError(t, stream.CloseSend())

	reply := dynamicpb.NewMessage(chat)
	require.NoError(t, stream.RecvMsg(reply))
	require.Equal(t, "pong", reply.Get(chat.Fields().ByName("text")).String())
}

func frameBody(t *testing.T, frame [2]any) []byte {
	t.Helper()

	payload, ok := frame[1].([]byte)
	require.True(t, ok)

	return payload
}
