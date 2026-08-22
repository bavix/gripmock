package mcp

import "context"

type ToolHandler func(context.Context, map[string]any) (map[string]any, error)

func DispatchTool(
	ctx context.Context,
	name string,
	args map[string]any,
	handlers map[string]ToolHandler,
) (map[string]any, error, bool) {
	handler, ok := handlers[name]
	if !ok {
		return map[string]any{}, nil, false
	}

	result, err := handler(ctx, args)

	return result, err, true
}
