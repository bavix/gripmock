package mcp

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
