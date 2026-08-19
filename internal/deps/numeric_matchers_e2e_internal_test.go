package deps

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const bigCount = 9007199254740993

func callWithCount(
	t *testing.T,
	srv *e2eServer,
	in, out protoreflect.MessageDescriptor,
	name string,
	count int64,
) (string, error) {
	t.Helper()

	conn := srv.dial(t)
	request := dynamicpb.NewMessage(in)

	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString(name))
	request.Set(in.Fields().ByName("count"), protoreflect.ValueOfInt64(count))

	reply := dynamicpb.NewMessage(out)
	if err := conn.Invoke(t.Context(), "/e2e.Greeter/SayHello", request, reply); err != nil {
		return "", err
	}

	return reply.Get(out.Fields().ByName("message")).String(), nil
}

func numericStub(name string, count int64, message string) map[string]any {
	return map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": name, "count": count}},
		"output":  map[string]any{"data": map[string]any{"message": message}},
	}
}

func TestNumericMatchersWorkFromEveryChannel(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)

	stubDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(stubDir, "file.json"), []byte(`{
		"service": "e2e.Greeter",
		"method": "SayHello",
		"input": {"equals": {"name": "FromFile", "count": 9007199254740993}},
		"output": {"data": {"message": "file ok"}}
	}`), 0o600))

	srv := startConfigured(t, protoPath, e2eOptions{stubPath: stubDir})

	srv.putStub(t, numericStub("FromRest", bigCount, "rest ok"), "")

	created := srv.mcpCall(t, "stubs_upsert", map[string]any{
		"stubs": numericStub("FromMCP", bigCount, "mcp ok"),
	})
	ids, ok := created["ids"].([]any)
	require.True(t, ok)
	require.Len(t, ids, 1)

	in, out := compileE2EDescriptors(t, protoPath)

	for _, tc := range []struct {
		name  string
		reply string
	}{
		{"FromFile", "file ok"},
		{"FromRest", "rest ok"},
		{"FromMCP", "mcp ok"},
	} {
		message, err := callWithCount(t, srv, in, out, tc.name, bigCount)
		require.NoErrorf(t, err, "stub %s must match", tc.name)
		require.Equal(t, tc.reply, message)
	}

	_, err := callWithCount(t, srv, in, out, "FromRest", bigCount-1)
	require.Error(t, err, "a neighbouring value must not match")
}

func TestFieldMaskDoesNotRoundOtherFields(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	srv := startConfigured(t, protoPath, e2eOptions{})

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"contains": map[string]any{"count": bigCount}},
		"output":  map[string]any{"data": map[string]any{"message": "big ok"}},
	}, "")

	plain := connectJSON(t, srv, "SayHello", `{"name":"A","count":9007199254740993}`)
	require.Equal(t, http.StatusOK, plain.status, string(plain.body))

	masked := connectJSON(t, srv, "SayHello",
		`{"name":"A","count":9007199254740993,"mask":{"paths":["name","count"]}}`)
	require.Equal(t, http.StatusOK, masked.status, string(masked.body))
	require.JSONEq(t, `{"message":"big ok"}`, string(masked.body))

	stringMask := connectJSON(t, srv, "SayHello",
		`{"name":"A","count":9007199254740993,"mask":"name,count"}`)
	require.Equal(t, http.StatusOK, stringMask.status, string(stringMask.body))
}

func TestValidateEchoesLargeIntegersExactly(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	srv := startConfigured(t, protoPath, e2eOptions{})

	payload, err := json.Marshal([]map[string]any{numericStub("Echo", bigCount, "ok")})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		srv.restURL+"/api/stubs/validate", bytes.NewReader(payload))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	require.Contains(t, string(body), "9007199254740993")
	require.NotContains(t, string(body), "9007199254740992")
}
