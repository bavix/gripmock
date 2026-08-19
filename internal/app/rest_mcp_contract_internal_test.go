package app

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func advertisedToolNames(t *testing.T) []string {
	t.Helper()

	tools := mcpusecase.ListRuntimeTools()
	names := make([]string, 0, len(tools))

	for _, tool := range tools {
		name, ok := tool["name"].(string)
		require.True(t, ok)

		names = append(names, name)
	}

	return names
}

func TestEveryAdvertisedToolIsBound(t *testing.T) {
	t.Parallel()

	server, err := NewRestServer(t.Context(), stuber.NewBudgerigar(), &mockExtender{}, history.NewMemoryStore(0), nil, nil, nil)
	require.NoError(t, err)

	handlers := mcpToolHandlers(server)

	for _, name := range advertisedToolNames(t) {
		require.Containsf(t, handlers, name, "tool %s is advertised without a handler", name)
	}

	advertised := make(map[string]struct{}, len(handlers))
	for _, name := range advertisedToolNames(t) {
		advertised[name] = struct{}{}
	}

	for name := range handlers {
		require.Containsf(t, advertised, name, "tool %s is bound but never advertised", name)
	}
}

func mcpToolFixtures() map[string]map[string]any {
	return map[string]map[string]any{
		mcpusecase.ToolHealthLiveness:   {},
		mcpusecase.ToolHealthReadiness:  {},
		mcpusecase.ToolHealthStatus:     {},
		mcpusecase.ToolDashboard:        {},
		mcpusecase.ToolOverview:         {},
		mcpusecase.ToolInfo:             {},
		mcpusecase.ToolSessionsList:     {},
		mcpusecase.ToolGripmockInfo:     {},
		mcpusecase.ToolReflectInfo:      {},
		mcpusecase.ToolReflectSources:   {},
		mcpusecase.ToolDescriptorsList:  {},
		mcpusecase.ToolServicesList:     {},
		mcpusecase.ToolServicesGet:      {"serviceID": "svc.Service"},
		mcpusecase.ToolServicesMethods:  {"serviceID": "svc.Service"},
		mcpusecase.ToolServicesMethod:   {"serviceID": "svc.Service", "methodID": "Method"},
		mcpusecase.ToolServicesDelete:   {"serviceID": "svc.Service"},
		mcpusecase.ToolHistoryList:      {},
		mcpusecase.ToolHistoryErrors:    {},
		mcpusecase.ToolHistoryPurge:     {},
		mcpusecase.ToolVerifyCalls:      {"service": "svc.Service", "method": "Method", "expectedCount": 0},
		mcpusecase.ToolDebugCall:        {"service": "svc.Service"},
		mcpusecase.ToolSchemaStub:       {},
		mcpusecase.ToolStubsList:        {},
		mcpusecase.ToolStubsGet:         {"id": "11111111-1111-1111-1111-111111111111"},
		mcpusecase.ToolStubsDelete:      {"id": "11111111-1111-1111-1111-111111111111"},
		mcpusecase.ToolStubsBatchDelete: {"ids": []any{"11111111-1111-1111-1111-111111111111"}},
		mcpusecase.ToolStubsPurge:       {},
		mcpusecase.ToolStubsSearch:      {"service": "svc.Service", "method": "Method", "data": map[string]any{"id": "1"}},
		mcpusecase.ToolStubsInspect:     {"service": "svc.Service", "method": "Method"},
		mcpusecase.ToolStubsUsed:        {},
		mcpusecase.ToolStubsUnused:      {},
		mcpusecase.ToolMockCall:         {"service": "svc.Service", "method": "Method", "payload": map[string]any{"id": "1"}},
		mcpusecase.ToolStubsUpsert:      {"stubs": stubFixture()},
		mcpusecase.ToolStubsValidate:    {"stubs": stubFixture()},
		mcpusecase.ToolDescriptorsAdd:   {"descriptorSetBase64": descriptorSetFixture()},
	}
}

func descriptorSetFixture() string {
	file := &descriptorpb.FileDescriptorProto{
		Name:    new("contract.proto"),
		Package: new("contract"),
		Syntax:  new("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: new("Ping"),
		}},
	}

	encoded, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}})
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(encoded)
}

func stubFixture() map[string]any {
	return map[string]any{
		"service": "svc.Service",
		"method":  "Method",
		"input":   map[string]any{"equals": map[string]any{"id": "1"}},
		"output":  map[string]any{"data": map[string]any{"ok": true}},
	}
}

func (s *RestServerTestSuite) TestEveryToolAnswers() {
	fixtures := mcpToolFixtures()

	for i, name := range advertisedToolNames(s.T()) {
		args, ok := fixtures[name]
		s.Require().Truef(ok, "tool %s has no argument fixture", name)

		response := s.mcpToolCall(s.server, i+1, name, args)

		s.Require().NotContainsf(mcpEnvelopeText(s.T(), response), "unknown tool", "tool %s is not wired", name)
		s.Require().NotContainsf(mcpEnvelopeText(s.T(), response), "method not found", "tool %s is not routed", name)
	}
}

func mcpEnvelopeText(t *testing.T, envelope map[string]any) string {
	t.Helper()

	if failure, ok := envelope["error"].(map[string]any); ok {
		message, _ := failure["message"].(string)

		return message
	}

	result, ok := envelope["result"].(map[string]any)
	if !ok {
		return ""
	}

	content, ok := result["content"].([]any)
	if !ok {
		return ""
	}

	var text strings.Builder

	for _, entry := range content {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		if value, ok := item["text"].(string); ok {
			text.WriteString(value)
		}
	}

	return text.String()
}

func (s *RestServerTestSuite) TestToolsRejectMalformedArguments() {
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"missing required", mcpusecase.ToolServicesGet, map[string]any{}, "serviceID"},
		{"uuid expected", mcpusecase.ToolStubsGet, map[string]any{"id": "not-a-uuid"}, "UUID"},
		{"uuid list expected", mcpusecase.ToolStubsBatchDelete, map[string]any{"ids": "one"}, "ids"},
		{"stub payload expected", mcpusecase.ToolStubsUpsert, map[string]any{"stubs": "text"}, "stubs"},
		{"base64 expected", mcpusecase.ToolDescriptorsAdd, map[string]any{"descriptorSetBase64": "!!!"}, "descriptorSetBase64"},
		{"negative limit", mcpusecase.ToolHistoryList, map[string]any{"limit": -1}, "limit"},
		{"unknown tool", "no_such_tool", map[string]any{}, "unknown tool"},
	}

	for i, tc := range cases {
		response := s.mcpToolCall(s.server, i+1, tc.tool, tc.args)
		text := mcpEnvelopeText(s.T(), response)

		s.Require().Containsf(text, tc.want, "%s: %v", tc.name, response)
	}
}

func TestDecodeToolArgumentsKeepsLargeIntegersExact(t *testing.T) {
	t.Parallel()

	args, err := decodeToolArguments([]byte(`{"count": 9007199254740993, "ratio": 1.5, "name": "x"}`))
	require.NoError(t, err)
	require.Equal(t, json.Number("9007199254740993"), args["count"])
	require.Equal(t, json.Number("1.5"), args["ratio"])
	require.Equal(t, "x", args["name"])

	empty, err := decodeToolArguments(nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	_, err = decodeToolArguments([]byte(`"not an object"`))
	require.Error(t, err)
}
