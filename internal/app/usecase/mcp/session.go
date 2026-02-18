package mcp

import (
	"net/http"

	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

func ApplyTransportSession(r *http.Request, toolName string, args map[string]any) map[string]any {
	if !ToolUsesSession(toolName) {
		return args
	}

	if args == nil {
		args = make(map[string]any)
	}

	if _, ok := args["session"]; ok {
		return args
	}

	if sessionID := session.FromRequest(r); sessionID != "" {
		args["session"] = sessionID
	}

	return args
}

func ToolUsesSession(toolName string) bool {
	switch toolName {
	case ToolHistoryList, ToolHistoryErrors, ToolDebugCall:
		return true
	case ToolStubsUpsert, ToolStubsList, ToolStubsPurge, ToolStubsSearch, ToolStubsUsed, ToolStubsUnused:
		return true
	default:
		return false
	}
}
