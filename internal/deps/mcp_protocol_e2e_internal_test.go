package deps

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func mcpRequest(t *testing.T, srv *e2eServer, payload map[string]any) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.restURL+"/api/mcp", bytes.NewReader(encoded))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	var envelope map[string]any

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))

	return envelope
}

func TestMCPHandshakeAndToolList(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	srv := startConfigured(t, protoPath, e2eOptions{})

	initialized := mcpRequest(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "e2e", "version": "0"},
		},
	})

	result, ok := initialized["result"].(map[string]any)
	require.True(t, ok, initialized)
	require.NotEmpty(t, result["protocolVersion"])

	serverInfo, ok := result["serverInfo"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, serverInfo["name"])

	listed := mcpRequest(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})

	listResult, ok := listed["result"].(map[string]any)
	require.True(t, ok, listed)

	tools, ok := listResult["tools"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, tools)

	names := make(map[string]struct{}, len(tools))

	for _, tool := range tools {
		entry, ok := tool.(map[string]any)
		require.True(t, ok)

		name, ok := entry["name"].(string)
		require.True(t, ok)

		names[name] = struct{}{}

		_, ok = entry["inputSchema"].(map[string]any)
		require.Truef(t, ok, "tool %s must publish an input schema", name)
	}

	for _, expected := range []string{"stubs_upsert", "history_list", "history_purge", "mock_call", "verify_calls"} {
		require.Containsf(t, names, expected, "tool %s must be advertised", expected)
	}
}

func TestMCPRejectsUnknownToolAndBadArguments(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	srv := startConfigured(t, protoPath, e2eOptions{})

	unknown := mcpRequest(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": "no_such_tool", "arguments": map[string]any{}},
	})
	require.True(t, mcpFailed(t, unknown), unknown)

	missingArgs := mcpRequest(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params":  map[string]any{"name": "mock_call", "arguments": map[string]any{"service": "e2e.Greeter"}},
	})
	require.True(t, mcpFailed(t, missingArgs), missingArgs)

	resources := mcpRequest(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "resources/list",
	})

	resourceResult, ok := resources["result"].(map[string]any)
	require.True(t, ok, resources)
	require.Empty(t, resourceResult["resources"])
}

func mcpFailed(t *testing.T, envelope map[string]any) bool {
	t.Helper()

	if _, ok := envelope["error"]; ok {
		return true
	}

	result, ok := envelope["result"].(map[string]any)
	if !ok {
		return false
	}

	failed, _ := result["isError"].(bool)

	return failed
}

func TestConcurrentCallsRespectTheCallBudget(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	srv := startConfigured(t, protoPath, e2eOptions{})

	const (
		budget  = 5
		callers = 40
	)

	srv.putStub(t, map[string]any{
		"service": "e2e.Greeter",
		"method":  "SayHello",
		"input":   map[string]any{"equals": map[string]any{"name": "Race"}},
		"output":  map[string]any{"data": map[string]any{"message": "served"}},
		"options": map[string]any{"times": budget},
	}, "")

	var (
		served   atomic.Int64
		rejected atomic.Int64
		failures atomic.Int64
		wg       sync.WaitGroup
	)

	wg.Add(callers)

	for range callers {
		go func() {
			defer wg.Done()

			status, _, err := rawConnectCall(t.Context(), srv, `{"name":"Race"}`, "")
			switch {
			case err != nil:
				failures.Add(1)
			case status == http.StatusOK:
				served.Add(1)
			case status == http.StatusNotFound:
				rejected.Add(1)
			}
		}()
	}

	wg.Wait()

	require.Zero(t, failures.Load())
	require.EqualValues(t, budget, served.Load(), "a times budget must not be overspent under concurrency")
	require.EqualValues(t, callers-budget, rejected.Load())
}

func TestConcurrentSessionsStayIsolated(t *testing.T) { //nolint:paralleltest // boots real servers
	protoPath := writeE2EProto(t)
	srv := startConfigured(t, protoPath, e2eOptions{})

	const sessions = 8

	for i := range sessions {
		session := sessionName(i)

		srv.putStub(t, map[string]any{
			"service": "e2e.Greeter",
			"method":  "SayHello",
			"input":   map[string]any{"equals": map[string]any{"name": "Shared"}},
			"output":  map[string]any{"data": map[string]any{"message": session}},
		}, session)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		replies []string
	)

	wg.Add(sessions)

	for i := range sessions {
		go func() {
			defer wg.Done()

			session := sessionName(i)

			for range 5 {
				status, body, err := rawConnectCall(t.Context(), srv, `{"name":"Shared"}`, session)

				reply := describeReply(session, status, body, err)

				mu.Lock()

				replies = append(replies, reply)

				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	require.Len(t, replies, sessions*5)

	for _, reply := range replies {
		require.True(t, strings.HasPrefix(reply, "ok "), reply)
	}
}

func rawConnectCall(ctx context.Context, srv *e2eServer, payload, session string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://"+srv.gatewayAddr+"/e2e.Greeter/SayHello", strings.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")

	if session != "" {
		req.Header.Set("X-Gripmock-Session", session)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, body, nil
}

func describeReply(session string, status int, body []byte, err error) string {
	if err != nil {
		return "error " + err.Error()
	}

	if status != http.StatusOK {
		return "status " + strconv.Itoa(status) + " " + string(body)
	}

	if !strings.Contains(string(body), `"`+session+`"`) {
		return "wrong session in " + string(body)
	}

	return "ok " + session
}

func sessionName(i int) string {
	return "session-" + strconv.Itoa(i)
}
