package mcp

type ToolHandler func(map[string]any) (map[string]any, error)

func DispatchTool(name string, args map[string]any, handlers map[string]ToolHandler) (map[string]any, error, bool) {
	handler, ok := handlers[name]
	if !ok {
		return map[string]any{}, nil, false
	}

	result, err := handler(args)

	return result, err, true
}
