package host

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/agent"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewMcpHost(_ *McpHostOptions) (McpHost, error) {

	client := mcp.NewClient(&mcp.Implementation{Name: "Remote MCP Host Client", Version: "0.1.0"}, nil)

	return McpHost{
		sessions:      make(map[string]*mcp.ClientSession),
		defaultClient: client,
		opts:          nil,
	}, nil
}

func (h *McpHost) GetClient(ctx context.Context, opts *ClientOptions) (agent.McpClient, error) {
	if opts == nil {
		opts = &ClientOptions{}
	}
	return HostMcpClient{host: h, opts: opts}, nil
}

// Opens MCP sessions with servers for this host.
// If a client is not specified, the host's default client is used.
func (h *McpHost) AddSessionsFromConfig(ctx context.Context, config io.Reader, client *mcp.Client) error {
	if client == nil {
		client = h.defaultClient
	}
	sessions, err := loadSessionsFromConfig(client, ctx, config)
	if err != nil {
		return err
	}

	for name, session := range sessions {
		if _, ok := h.sessions[name]; ok {
			return fmt.Errorf("server name conflict: %s", name)
		}
		h.sessions[name] = session
	}
	return nil
}

func (h *McpHost) ListServerNames() []string {
	keys := make([]string, 0, len(h.sessions))
	for k := range h.sessions {
		keys = append(keys, k)
	}
	return keys
}

// Gets a session for an MCP server with a particular name
func (h *McpHost) GetSession(ctx context.Context, name string) (*mcp.ClientSession, error) {
	session, ok := h.sessions[name]
	if !ok {
		return session, fmt.Errorf("invalid ClientSession name: %s", name)
	}
	return session, nil
}

func (h *McpHost) Tools(ctx context.Context) iter.Seq2[*agent.ServerTool, error] {
	return func(yield func(*agent.ServerTool, error) bool) {
		for _, serverName := range h.ListServerNames() {
			session, err := h.GetSession(ctx, serverName)
			if err != nil {
				yield(nil, err)
				return
			}
			for tool, err := range session.Tools(ctx, nil) {
				if err != nil {
					yield(nil, err)
					return
				}
				yield(&agent.ServerTool{
					ServerName: serverName,
					Tool:       *tool,
				}, nil)
			}
		}
	}
}

// Lists all tools for a server that has an open session with this host
func (h *McpHost) ListToolsOnServer(ctx context.Context, serverName string) ([]mcp.Tool, error) {
	session, err := h.GetSession(ctx, serverName)

	if err != nil {
		return nil, err
	}
	if session.InitializeResult().Capabilities.Tools == nil {
		return []mcp.Tool{}, nil
	}

	var tools []mcp.Tool

	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, fmt.Errorf("fetching tools for `%s`: %s", serverName, err)
		}
		tools = append(tools, *tool)
	}

	return tools, nil
}

func (hmc HostMcpClient) CallTool(ctx context.Context, toolRequest *agent.ServerToolRequest) (*api.ToolUsePart, error) {

	toolRequestId := api.ToolId{ServerName: toolRequest.ServerName, Name: toolRequest.Name}
	var config *api.ToolConfig = nil

	for _, toolConfig := range hmc.opts.ToolConfigs {
		if toolConfig.ToolId == toolRequestId {
			config = toolConfig
			break
		}
	}

	if hmc.opts.OnlyUseConfiguredTools && config == nil {
		return nil, fmt.Errorf("invalid ToolId")
	} else if config == nil {
		config = &api.ToolConfig{ToolId: toolRequestId, ToolPatch: api.ToolPatch{Input: nil}}
	}

	session, err := hmc.host.GetSession(ctx, toolRequest.ServerName)
	if err != nil {
		return nil, fmt.Errorf("could not connect to session '%s': %s", toolRequest.ServerName, err)
	}

	res, err := session.CallTool(ctx, &patchToolRequest(toolRequest, config.ToolPatch).CallToolParams)
	if err != nil {
		return nil, fmt.Errorf("error calling tool '%s': %s", toolRequest.Name, err)
	}

	toolUsePart := api.NewToolUsePart(
		toolRequest.CallToolParams.Arguments,
		*res,
		api.ToolId{
			Name:       toolRequest.Name,
			ServerName: toolRequest.ServerName})

	return &toolUsePart, nil
}

func (hmc HostMcpClient) ListTools(ctx context.Context) ([]*agent.ServerTool, error) {

	var serverTools []*agent.ServerTool

	for tool, err := range hmc.host.Tools(ctx) {
		if err != nil {
			return serverTools, err
		}

		found := false
		for _, toolConfig := range hmc.opts.ToolConfigs {
			if tool.Name == toolConfig.ToolId.Name && tool.ServerName == toolConfig.ToolId.ServerName {
				serverTools = append(serverTools, tool)
				found = true
				break
			}
		}
		if !found && !hmc.opts.OnlyUseConfiguredTools {
			serverTools = append(serverTools, tool)
		}

	}

	return serverTools, nil
}

func patchToolRequest(toolRequest *agent.ServerToolRequest, patch api.ToolPatch) *agent.ServerToolRequest {
	patchedReq := *toolRequest

	if patch.Input == nil {
		return &patchedReq
	}

	if hash, ok := patchedReq.Arguments.(map[string]any); ok {
		for key, val := range patch.Input {
			hash[key] = val
		}
	} else {
		patchedReq.Arguments = patch.Input
	}
	return &patchedReq
}

func patchTool(tool *mcp.Tool, patch api.ToolPatch) *mcp.Tool {
	patchedTool := *tool

	if patch.Input == nil {
		return &patchedTool
	}

	if hash, ok := patchedTool.InputSchema.(map[string]any); ok {
		for key := range patch.Input {
			delete(hash, key)
		}
	} else {
		log.Printf("Warning: could not patch InputSchema because it was not map[string]any")
	}
	return &patchedTool
}

func loadSessionsFromConfig(client *mcp.Client, ctx context.Context, r io.Reader) (map[string]*mcp.ClientSession, error) {
	scanner := bufio.NewScanner(r)

	sessions := make(map[string]*mcp.ClientSession)

	for scanner.Scan() {
		sessionWithName, err := sessionFromLine(client, ctx, scanner.Text())
		if err != nil {
			return sessions, err
		}

		if _, exists := sessions[sessionWithName.sessionName]; exists {
			return sessions, fmt.Errorf("server name conflict: %s", sessionWithName.sessionName)
		}
		sessions[sessionWithName.sessionName] = sessionWithName.session
	}

	return sessions, nil
}

var stdioRegex = regexp.MustCompile(`^!\[([^\]]+)\](\[(\w[\w\d-_]*)\])?\s*(\S+)\s*(.*)$`)
var httpRegex = regexp.MustCompile(`^>\[(\w[\w\d-_]*)\]\s*(http://.+)$`)

func sessionFromLine(client *mcp.Client, ctx context.Context, line string) (clientSessionWithName, error) {
	if matches := stdioRegex.FindStringSubmatch(line); len(matches) > 0 {
		command := matches[len(matches)-2]
		arguments := splitIntoWords(matches[len(matches)-1])
		cmd := exec.Command(command, arguments...)
		cmd.Dir = matches[1]
		sessionName := command

		if matches[3] != "" {
			sessionName = matches[3]
		}

		transport := &mcp.CommandTransport{Command: cmd}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return clientSessionWithName{}, err
		}
		return clientSessionWithName{
			session:     session,
			sessionName: sessionName,
		}, nil

	} else if matches := httpRegex.FindStringSubmatch(line); len(matches) > 0 {
		sessionName := matches[1]
		url := matches[2]

		session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url}, nil)
		if err != nil {
			return clientSessionWithName{}, err
		}

		return clientSessionWithName{
			session:     session,
			sessionName: sessionName,
		}, nil
	}

	return clientSessionWithName{}, fmt.Errorf("invalid line in config: %s", line)
}

var whiteSpaceRegex = regexp.MustCompile(`\s+`)

func splitIntoWords(line string) []string {
	words := whiteSpaceRegex.Split(strings.TrimSpace(line), -1)
	return words
}
