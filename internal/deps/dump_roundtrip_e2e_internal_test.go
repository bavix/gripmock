package deps

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func dumpStubs(t *testing.T, srv *e2eServer, format string) string {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.restURL+"/api/stubs", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var stubs []*stuber.Stub

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stubs))
	require.NotEmpty(t, stubs)

	outDir := t.TempDir()

	files, err := stuber.DumpToDir(outDir, stubs, format)
	require.NoError(t, err)
	require.Positive(t, files)

	return outDir
}

func seedDumpStubs(t *testing.T, srv *e2eServer) {
	t.Helper()

	srv.putStub(t, map[string]any{
		"service":  "e2e.Greeter",
		"method":   "SayHello",
		"headers":  map[string]any{"equals": map[string]any{"x-tenant": "acme"}},
		"input":    map[string]any{"equals": map[string]any{"name": "Gated"}},
		"output":   map[string]any{"data": map[string]any{"message": "gated reply"}, "headers": map[string]any{"x-shard": "eu-1"}},
		"priority": 7,
	}, "")

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"matches": map[string]any{"name": "^Regex.*"}},
		"output":  map[string]any{"data": map[string]any{"message": "matched by regex"}},
	}, "")

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Boom"}},
		"output": map[string]any{
			"error": "nope",
			"code":  7,
			"details": []any{map[string]any{
				"type":   "type.googleapis.com/google.rpc.ErrorInfo",
				"reason": "DENIED",
				"domain": "e2e",
			}},
		},
	}, "")

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "Watch",
		"input":   map[string]any{"equals": map[string]any{"topic": "orders"}},
		"output": map[string]any{"stream": []any{
			map[string]any{"payload": "one"},
			map[string]any{"payload": "two"},
		}},
	}, "")

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "Collect",
		"inputs": []any{
			map[string]any{"equals": map[string]any{"part": "a"}},
			map[string]any{"equals": map[string]any{"part": "b"}},
		},
		"output": map[string]any{"data": map[string]any{"digest": "ab"}},
	}, "")
}

func assertDumpedBehaviour(t *testing.T, srv *e2eServer) {
	t.Helper()

	gated := connectJSON(t, srv, "SayHello", `{"name":"Gated"}`, [2]string{"X-Tenant", "acme"})
	require.Equal(t, http.StatusOK, gated.status, string(gated.body))
	require.JSONEq(t, `{"message":"gated reply"}`, string(gated.body))
	require.Equal(t, "eu-1", gated.header.Get("X-Shard"))

	ungated := connectJSON(t, srv, "SayHello", `{"name":"Gated"}`)
	require.Equal(t, http.StatusNotFound, ungated.status, "the header matcher must survive the dump")

	regex := connectJSON(t, srv, "SayHello", `{"name":"RegexValue"}`)
	require.Equal(t, http.StatusOK, regex.status, string(regex.body))
	require.JSONEq(t, `{"message":"matched by regex"}`, string(regex.body))

	failing := connectJSON(t, srv, "SayHello", `{"name":"Boom"}`)
	require.Equal(t, http.StatusForbidden, failing.status, string(failing.body))
	require.Contains(t, string(failing.body), `"reason":"DENIED"`)

	stream := gatewayPost(t, "http://"+srv.gatewayAddr+"/e2e.Greeter/Watch",
		"application/connect+json", string(buildWebFrame([]byte(`{"topic":"orders"}`))),
		[2]string{"Connect-Protocol-Version", "1"})
	require.Equal(t, http.StatusOK, stream.status, string(stream.body))

	frames := readFrames(t, stream.body)
	require.Len(t, frames, 3)
	require.Contains(t, string(frameBody(t, frames[0])), "one")
	require.Contains(t, string(frameBody(t, frames[1])), "two")

	chunk, summary := messagePair(t, srv.protoPath, "Chunk", "Summary")
	conn := srv.dial(t)

	client, err := conn.NewStream(t.Context(), clientStreamDesc(), "/e2e.Greeter/Collect")
	require.NoError(t, err)

	sendChunks(t, client, chunk, "a", "b")

	reply := newMessage(summary)
	require.NoError(t, client.RecvMsg(reply))
	require.Equal(t, "ab", reply.Get(summary.Fields().ByName("digest")).String())

	missing, err := conn.NewStream(t.Context(), clientStreamDesc(), "/e2e.Greeter/Collect")
	require.NoError(t, err)

	sendChunks(t, missing, chunk, "b", "a")
	require.Equal(t, codes.NotFound, status.Code(missing.RecvMsg(newMessage(summary))))
}

func TestDumpedStubsReloadIdentically(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)

	//nolint:paralleltest // each case boots its own servers
	for _, format := range []string{stuber.DumpFormatYAML, stuber.DumpFormatJSON} {
		t.Run(format, func(t *testing.T) {
			source := startConfigured(t, protoPath, e2eOptions{})
			seedDumpStubs(t, source)
			assertDumpedBehaviour(t, source)

			outDir := dumpStubs(t, source, format)

			reloaded := startConfigured(t, protoPath, e2eOptions{stubPath: outDir})
			assertDumpedBehaviour(t, reloaded)
		})
	}
}

func clientStreamDesc() *grpc.StreamDesc {
	return &grpc.StreamDesc{StreamName: "Collect", ClientStreams: true}
}

func newMessage(desc protoreflect.MessageDescriptor) *dynamicpb.Message {
	return dynamicpb.NewMessage(desc)
}

func sendChunks(t *testing.T, stream grpc.ClientStream, desc protoreflect.MessageDescriptor, parts ...string) {
	t.Helper()

	for _, part := range parts {
		msg := dynamicpb.NewMessage(desc)

		msg.Set(desc.Fields().ByName("part"), protoreflect.ValueOfString(part))
		require.NoError(t, stream.SendMsg(msg))
	}

	require.NoError(t, stream.CloseSend())
}
