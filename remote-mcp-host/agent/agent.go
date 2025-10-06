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
	// If GenerateResult.Stop is not set to true, the generate function
	// may be called again before the new message is finalized and sent
	// to the user
	Generate(context.Context, []api.Message, *GenerateOptions) (*GenerateResult, error)
}

type GenerateOptions struct {

	// The list of tools that are available for the Agent.
	Tools []*ServerTool

	// The parts already generated for the response
	// If this is non-empty, then the language model should continue
	// to build on the parts.
	GeneratedParts []api.UnionPart
}

type ServerTool struct {
	ServerName string
	mcp.Tool
}

type GenerateResult struct {
	Parts        []api.UnionPart
	ToolRequests []ToolRequest
	Stop         bool
}

type ToolRequest struct {
	ServerName string
	mcp.CallToolParams
}
