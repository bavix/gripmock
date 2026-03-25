package mcp

import (
	"net/http"

	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

func ApplyTransportSession(r *http.Request, toolName string, args map[string]any) map[string]any {
	return ApplySession(toolName, args, muxmiddleware.FromRequest(r))
}

func ApplySession(toolName string, args map[string]any, sessionID string) map[string]any {
	if !ToolUsesSession(toolName) {
		return args
	}

	if args == nil {
		args = make(map[string]any)
	}

	if _, ok := args["session"]; ok {
		return args
	}

	if sessionID != "" {
		args["session"] = sessionID
	}

	return args
}

func ToolUsesSession(toolName string) bool {
	switch toolName {
	case ToolDashboard, ToolOverview, ToolInfo, ToolHistoryList, ToolHistoryErrors, ToolVerifyCalls, ToolDebugCall:
		return true
	case ToolStubsUpsert, ToolStubsList, ToolStubsPurge, ToolStubsSearch, ToolStubsInspect, ToolStubsUsed, ToolStubsUnused:
		return true
	default:
		return false
	}
}
