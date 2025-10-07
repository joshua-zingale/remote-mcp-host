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
	//
	// If GenerateResult.Continue is set to true, the generate function
	// may be called again before the new message is finalized and sent
	// to the user
	Generate(context.Context, []api.Message, McpClient, *GenerateOptions) (*GenerateResult, error)
}

type GenerateOptions struct {
	_ bool
}

type McpClient interface {
	CallTool(context.Context, *ServerToolRequest) (*api.ToolUsePart, error)
	ListTools(context.Context) ([]*ServerTool, error)
}

type ServerTool struct {
	ServerName string
	mcp.Tool
}

type GenerateResult struct {
	Parts []api.UnionPart
}

type ServerToolRequest struct {
	ServerName string
	mcp.CallToolParams
}
