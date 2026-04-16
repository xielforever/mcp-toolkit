package mcp

import "context"

type ToolHandler func(ctx context.Context, input map[string]any) (any, error)

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     ToolHandler
}

