package mcp

const (
	ToolDescriptorsAdd  = "descriptors.add"
	ToolDescriptorsList = "descriptors.list"
	ToolServicesList    = "services.list"
	ToolServicesDelete  = "services.delete"
	ToolHistoryList     = "history.list"
	ToolHistoryErrors   = "history.errors"
	ToolDebugCall       = "debug.call"
)

func ListTools() []map[string]any {
	return []map[string]any{
		descriptorsAddTool(),
		descriptorsListTool(),
		servicesListTool(),
		servicesDeleteTool(),
		historyListTool(),
		historyErrorsTool(),
		debugCallTool(),
	}
}

func descriptorsAddTool() map[string]any {
	return map[string]any{
		"name":        ToolDescriptorsAdd,
		"description": "Register a protobuf FileDescriptorSet encoded as base64",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"descriptorSetBase64",
			},
			"properties": map[string]any{
				"descriptorSetBase64": map[string]any{
					"type":        "string",
					"description": "Binary FileDescriptorSet encoded with base64",
				},
			},
		},
	}
}

func descriptorsListTool() map[string]any {
	return map[string]any{
		"name":        ToolDescriptorsList,
		"description": "List service IDs registered through dynamic descriptors",
		"inputSchema": map[string]any{"type": "object"},
	}
}

func servicesListTool() map[string]any {
	return map[string]any{
		"name":        ToolServicesList,
		"description": "List all currently available gRPC services",
		"inputSchema": map[string]any{"type": "object"},
	}
}

func servicesDeleteTool() map[string]any {
	return map[string]any{
		"name":        ToolServicesDelete,
		"description": "Delete a service previously registered via dynamic descriptors",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"serviceID",
			},
			"properties": map[string]any{
				"serviceID": map[string]any{
					"type": "string",
				},
			},
		},
	}
}

func historyListTool() map[string]any {
	return map[string]any{
		"name":        ToolHistoryList,
		"description": "List recent gRPC call history for debugging",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"service": map[string]any{"type": "string"},
				"method":  map[string]any{"type": "string"},
				"session": map[string]any{"type": "string"},
				"limit":   map[string]any{"type": "integer", "minimum": 0},
			},
		},
	}
}

func historyErrorsTool() map[string]any {
	return map[string]any{
		"name":        ToolHistoryErrors,
		"description": "List recent gRPC calls that ended with errors",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session": map[string]any{"type": "string"},
				"limit":   map[string]any{"type": "integer", "minimum": 0},
			},
		},
	}
}

func debugCallTool() map[string]any {
	return map[string]any{
		"name":        ToolDebugCall,
		"description": "Diagnose why a service/method call is failing",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"service",
			},
			"properties": map[string]any{
				"service":    map[string]any{"type": "string"},
				"method":     map[string]any{"type": "string"},
				"session":    map[string]any{"type": "string"},
				"limit":      map[string]any{"type": "integer", "minimum": 0},
				"stubsLimit": map[string]any{"type": "integer", "minimum": 0},
			},
		},
	}
}
