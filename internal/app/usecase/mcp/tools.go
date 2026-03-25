package mcp

const (
	ToolHealthLiveness  = "health.liveness"
	ToolHealthReadiness = "health.readiness"
	ToolHealthStatus    = "health.status"

	ToolDashboard = "dashboard.full"
	ToolOverview  = "dashboard.overview"
	ToolInfo      = "dashboard.info"

	ToolSessionsList   = "sessions.list"
	ToolGripmockInfo   = "gripmock.info"
	ToolReflectInfo    = "reflect.info"
	ToolReflectSources = "reflect.sources"

	ToolDescriptorsAdd  = "descriptors.add"
	ToolDescriptorsList = "descriptors.list"
	ToolServicesGet     = "services.get"
	ToolServicesMethods = "services.methods"
	ToolServicesMethod  = "services.method"
	ToolServicesList    = "services.list"
	ToolServicesDelete  = "services.delete"
	ToolHistoryList     = "history.list"
	ToolHistoryErrors   = "history.errors"
	ToolVerifyCalls     = "verify.calls"
	ToolDebugCall       = "debug.call"
	ToolSchemaStub      = "schema.stub"

	ToolStubsUpsert      = "stubs.upsert"
	ToolStubsList        = "stubs.list"
	ToolStubsGet         = "stubs.get"
	ToolStubsDelete      = "stubs.delete"
	ToolStubsBatchDelete = "stubs.batchDelete"
	ToolStubsPurge       = "stubs.purge"
	ToolStubsSearch      = "stubs.search"
	ToolStubsInspect     = "stubs.inspect"
	ToolStubsUsed        = "stubs.used"
	ToolStubsUnused      = "stubs.unused"
)

func ListTools() []map[string]any {
	return []map[string]any{
		healthLivenessTool(),
		healthReadinessTool(),
		healthStatusTool(),
		dashboardTool(),
		dashboardOverviewTool(),
		dashboardInfoTool(),
		sessionsListTool(),
		gripmockInfoTool(),
		reflectInfoTool(),
		reflectSourcesTool(),
		descriptorsAddTool(),
		descriptorsListTool(),
		servicesListTool(),
		servicesGetTool(),
		servicesMethodsTool(),
		servicesMethodTool(),
		servicesDeleteTool(),
		historyListTool(),
		historyErrorsTool(),
		verifyCallsTool(),
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
		stubsInspectTool(),
		stubsUsedTool(),
		stubsUnusedTool(),
	)
}

func newTool(name string, description string, inputSchema map[string]any) map[string]any {
	return map[string]any{
		"name":        name,
		"description": description,
		"inputSchema": inputSchema,
	}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object"}

	if len(required) > 0 {
		schema["required"] = required
	}

	if len(properties) > 0 {
		schema["properties"] = properties
	}

	return schema
}

func stringProp() map[string]any {
	return map[string]any{"type": "string"}
}

func nonNegativeIntegerProp() map[string]any {
	return map[string]any{"type": "integer", "minimum": 0}
}

func objectAnyProp() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": true}
}

func objectArrayAnyProp() map[string]any {
	return map[string]any{"type": "array", "items": objectAnyProp()}
}

func healthLivenessTool() map[string]any {
	return newTool(ToolHealthLiveness, "Get service liveness status", objectSchema(nil))
}

func healthReadinessTool() map[string]any {
	return newTool(ToolHealthReadiness, "Get service readiness status", objectSchema(nil))
}

func healthStatusTool() map[string]any {
	return newTool(ToolHealthStatus, "Get combined liveness and readiness status", objectSchema(nil))
}

func dashboardTool() map[string]any {
	return newTool(ToolDashboard, "Get full dashboard payload", objectSchema(map[string]any{"session": stringProp()}))
}

func dashboardOverviewTool() map[string]any {
	return newTool(ToolOverview, "Get lightweight dashboard overview", objectSchema(map[string]any{"session": stringProp()}))
}

func dashboardInfoTool() map[string]any {
	return newTool(ToolInfo, "Get runtime and build information", objectSchema(map[string]any{"session": stringProp()}))
}

func sessionsListTool() map[string]any {
	return newTool(ToolSessionsList, "List active non-empty sessions", objectSchema(nil))
}

func gripmockInfoTool() map[string]any {
	return newTool(ToolGripmockInfo, "Get GripMock runtime and capability metadata", objectSchema(nil))
}

func reflectInfoTool() map[string]any {
	return newTool(ToolReflectInfo, "Get reflection-related descriptor statistics", objectSchema(nil))
}

func reflectSourcesTool() map[string]any {
	return newTool(ToolReflectSources, "List runtime descriptor source paths with optional filtering", objectSchema(map[string]any{
		"kind":   map[string]any{"type": "string", "enum": []string{"all", "reflection", "dynamic"}},
		"offset": nonNegativeIntegerProp(),
		"limit":  nonNegativeIntegerProp(),
	}))
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
	return newTool(ToolDescriptorsList, "List service IDs registered through dynamic descriptors", objectSchema(nil))
}

func servicesListTool() map[string]any {
	return newTool(ToolServicesList, "List all currently available gRPC services", objectSchema(nil))
}

func servicesGetTool() map[string]any {
	return newTool(ToolServicesGet, "Get detailed service metadata", objectSchema(map[string]any{
		"serviceID": stringProp(),
	}, "serviceID"))
}

func servicesMethodsTool() map[string]any {
	return newTool(ToolServicesMethods, "List methods for a service", objectSchema(map[string]any{
		"serviceID": stringProp(),
	}, "serviceID"))
}

func servicesMethodTool() map[string]any {
	return newTool(ToolServicesMethod, "Get one service method metadata", objectSchema(map[string]any{
		"serviceID": stringProp(),
		"methodID":  stringProp(),
	}, "serviceID", "methodID"))
}

func servicesDeleteTool() map[string]any {
	return newTool(ToolServicesDelete, "Delete a service previously registered via dynamic descriptors", objectSchema(map[string]any{
		"serviceID": stringProp(),
	}, "serviceID"))
}

func historyListTool() map[string]any {
	return newTool(ToolHistoryList, "List recent gRPC call history for debugging", objectSchema(map[string]any{
		"service": stringProp(),
		"method":  stringProp(),
		"session": stringProp(),
		"limit":   nonNegativeIntegerProp(),
	}))
}

func historyErrorsTool() map[string]any {
	return newTool(ToolHistoryErrors, "List recent gRPC calls that ended with errors", objectSchema(map[string]any{
		"session": stringProp(),
		"limit":   nonNegativeIntegerProp(),
	}))
}

func debugCallTool() map[string]any {
	return newTool(ToolDebugCall, "Diagnose why a service/method call is failing", objectSchema(map[string]any{
		"service":    stringProp(),
		"method":     stringProp(),
		"session":    stringProp(),
		"limit":      nonNegativeIntegerProp(),
		"stubsLimit": nonNegativeIntegerProp(),
	}, "service"))
}

func verifyCallsTool() map[string]any {
	return newTool(ToolVerifyCalls, "Verify expected gRPC call count", objectSchema(map[string]any{
		"service":       stringProp(),
		"method":        stringProp(),
		"expectedCount": nonNegativeIntegerProp(),
		"session":       stringProp(),
	}, "service", "method", "expectedCount"))
}

func schemaStubTool() map[string]any {
	return newTool(ToolSchemaStub, "Return JSON Schema URL for stubs payload", objectSchema(nil))
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
	return newTool(ToolStubsList, "List stubs with optional filters", objectSchema(map[string]any{
		"service": stringProp(),
		"method":  stringProp(),
		"session": stringProp(),
		"limit":   nonNegativeIntegerProp(),
		"offset":  nonNegativeIntegerProp(),
	}))
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
	return newTool(ToolStubsPurge, "Delete all stubs or session-scoped stubs", objectSchema(map[string]any{"session": stringProp()}))
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

func stubsInspectTool() map[string]any {
	return newTool(ToolStubsInspect, "Inspect stub matching decision path", objectSchema(map[string]any{
		"id":      stringProp(),
		"service": stringProp(),
		"method":  stringProp(),
		"session": stringProp(),
		"headers": objectAnyProp(),
		"input":   objectArrayAnyProp(),
	}, "service", "method"))
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
