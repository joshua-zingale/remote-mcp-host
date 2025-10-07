package agent

import (
	"context"

	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Agent interface {
	// Completes text.
	// The messages should be ordered from oldest to newest.
	// The options may be null, in which case default values should be used.
	Act(context.Context, McpClient, []api.Message, *GenerateOptions) (*GenerateResult, error)
}

type GenerateOptions struct {
	_ bool
}

type GenerateResult struct {
	Message *api.Message
}

type McpClient interface {
	CallTool(context.Context, *ServerToolRequest) (*api.ToolUsePart, error)
	ListTools(context.Context) ([]*ServerTool, error)
}

type ServerTool struct {
	ServerName string
	mcp.Tool
}

type ServerToolRequest struct {
	ServerName string
	mcp.CallToolParams
}
