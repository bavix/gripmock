package mcp

const (
	ToolDescriptorsAdd  = "descriptors.add"
	ToolDescriptorsList = "descriptors.list"
	ToolServicesList    = "services.list"
	ToolServicesDelete  = "services.delete"
	ToolHistoryList     = "history.list"
	ToolHistoryErrors   = "history.errors"
	ToolDebugCall       = "debug.call"
	ToolSchemaStub      = "schema.stub"

	ToolStubsUpsert      = "stubs.upsert"
	ToolStubsList        = "stubs.list"
	ToolStubsGet         = "stubs.get"
	ToolStubsDelete      = "stubs.delete"
	ToolStubsBatchDelete = "stubs.batchDelete"
	ToolStubsPurge       = "stubs.purge"
	ToolStubsSearch      = "stubs.search"
	ToolStubsUsed        = "stubs.used"
	ToolStubsUnused      = "stubs.unused"
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
		schemaStubTool(),
	}
}

func ListRuntimeTools() []map[string]any {
	tools := append([]map[string]any{}, ListTools()...)

	return append(tools,
		stubsUpsertTool(),
		stubsListTool(),
		stubsGetTool(),
		stubsDeleteTool(),
		stubsBatchDeleteTool(),
		stubsPurgeTool(),
		stubsSearchTool(),
		stubsUsedTool(),
		stubsUnusedTool(),
	)
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

func schemaStubTool() map[string]any {
	return map[string]any{
		"name":        ToolSchemaStub,
		"description": "Return JSON Schema URL for stubs payload",
		"inputSchema": map[string]any{"type": "object"},
	}
}

func stubsUpsertTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsUpsert,
		"description": "Create or update one or many stubs",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"stubs",
			},
			"properties": map[string]any{
				"session": map[string]any{"type": "string"},
				"stubs": map[string]any{
					"description": "Stub object or array of stub objects",
					"oneOf": []map[string]any{
						{"type": "object", "additionalProperties": true},
						{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
					},
				},
			},
		},
	}
}

func stubsListTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsList,
		"description": "List stubs with optional filters",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"service": map[string]any{"type": "string"},
				"method":  map[string]any{"type": "string"},
				"session": map[string]any{"type": "string"},
				"limit":   map[string]any{"type": "integer", "minimum": 0},
				"offset":  map[string]any{"type": "integer", "minimum": 0},
			},
		},
	}
}

func stubsGetTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsGet,
		"description": "Get a stub by ID",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"id",
			},
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
		},
	}
}

func stubsDeleteTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsDelete,
		"description": "Delete a stub by ID",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"id",
			},
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
		},
	}
}

func stubsBatchDeleteTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsBatchDelete,
		"description": "Delete stubs by IDs",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"ids",
			},
			"properties": map[string]any{
				"ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
		},
	}
}

func stubsPurgeTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsPurge,
		"description": "Delete all stubs or session-scoped stubs",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session": map[string]any{"type": "string"},
			},
		},
	}
}

func stubsSearchTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsSearch,
		"description": "Search a matching stub by request payload",
		"inputSchema": map[string]any{
			"type": "object",
			"required": []string{
				"service",
				"method",
				"payload",
			},
			"properties": map[string]any{
				"service": map[string]any{"type": "string"},
				"method":  map[string]any{"type": "string"},
				"session": map[string]any{"type": "string"},
				"headers": map[string]any{"type": "object", "additionalProperties": true},
				"payload": map[string]any{"type": "object", "additionalProperties": true},
				"input": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "object", "additionalProperties": true},
				},
			},
		},
	}
}

func stubsUsedTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsUsed,
		"description": "List used stubs with optional filters",
		"inputSchema": stubsListTool()["inputSchema"],
	}
}

func stubsUnusedTool() map[string]any {
	return map[string]any{
		"name":        ToolStubsUnused,
		"description": "List unused stubs with optional filters",
		"inputSchema": stubsListTool()["inputSchema"],
	}
}
