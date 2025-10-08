package host

import (
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ClientOptions struct {
	ToolConfigs            []*api.ToolConfig
	OnlyUseConfiguredTools bool
}

type HostMcpClient struct {
	host *McpHost
	opts *ClientOptions
}

type clientSessionWithName struct {
	session     *mcp.ClientSession
	sessionName string
}

type McpHost struct {
	sessions      map[string]*mcp.ClientSession
	defaultClient *mcp.Client
	opts          *McpHostOptions
}

type McpHostOptions struct {
	_ bool
}
