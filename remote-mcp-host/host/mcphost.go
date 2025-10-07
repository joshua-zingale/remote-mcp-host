package host

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"os/exec"
	"regexp"
	"strings"

	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/agent"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type clientSessionWithName struct {
	session     *mcp.ClientSession
	sessionName string
}

type McpHost struct {
	sessions      map[string]*mcp.ClientSession
	defaultClient *mcp.Client
	agent         agent.Agent
	opts          *McpHostOptions
}

type McpHostOptions struct {
	_ bool
}

func NewMcpHost(agent agent.Agent, _ *McpHostOptions) (McpHost, error) {

	client := mcp.NewClient(&mcp.Implementation{Name: "Remote MCP Host Client", Version: "0.1.0"}, nil)

	return McpHost{
		sessions:      make(map[string]*mcp.ClientSession),
		defaultClient: client,
		agent:         agent,
		opts:          nil,
	}, nil
}

func (h *McpHost) allTools(ctx context.Context) ([]*api.ToolConfig, error) {
	var availableTools []*api.ToolConfig

	for tool, err := range h.Tools(ctx) {
		if err != nil {
			return availableTools, err
		}
		availableTools = append(availableTools, &api.ToolConfig{
			ToolId: api.ToolId{
				ServerName: tool.ServerName,
				Name:       tool.Name,
			}})
	}

	return availableTools, nil
}

func (h *McpHost) GetClientWithFeatures(ctx context.Context, opts *ClientFeatures) (agent.McpClient, error) {
	if opts == nil {
		opts = &ClientFeatures{}
	}

	if opts.AvailableTools == nil {
		allTools, err := h.allTools(ctx)
		if err != nil {
			return nil, err
		}
		opts.AvailableTools = allTools
	}
	return HostMcpClient{host: h, opts: opts}, nil
}

// Generates a new message from the input message history.
func (h *McpHost) Generate(ctx context.Context, messages []api.Message, opts *HostGenerateOptions) (*api.Message, error) {
	if opts == nil {
		opts = &HostGenerateOptions{}
	}

	client, err := h.GetClientWithFeatures(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %s", err)
	}

	res, err := h.agent.Generate(ctx, messages, client, &agent.GenerateOptions{})
	if err != nil {
		return nil, fmt.Errorf("generating response: %s", err)
	}

	return &api.Message{
		Parts: res.Parts,
		Role:  "model",
	}, nil
}

type HostGenerateOptions struct{}

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
func (h *McpHost) GetClientSession(ctx context.Context, name string) (*mcp.ClientSession, error) {
	session, ok := h.sessions[name]
	if !ok {
		return session, fmt.Errorf("invalid ClientSession name: %s", name)
	}
	return session, nil
}

func (h *McpHost) Tools(ctx context.Context) iter.Seq2[*agent.ServerTool, error] {
	return func(yield func(*agent.ServerTool, error) bool) {
		for _, serverName := range h.ListServerNames() {
			session, err := h.GetClientSession(ctx, serverName)
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
	session, err := h.GetClientSession(ctx, serverName)

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

type ClientFeatures struct {
	AvailableTools []*api.ToolConfig
}

type HostMcpClient struct {
	host *McpHost
	opts *ClientFeatures
}

func (hmc HostMcpClient) CallTool(ctx context.Context, toolRequest *agent.ServerToolRequest) (*api.ToolUsePart, error) {
	session, err := hmc.host.GetClientSession(ctx, toolRequest.ServerName)
	if err != nil {
		return nil, fmt.Errorf("could not connect to session '%s': %s", toolRequest.ServerName, err)
	}

	res, err := session.CallTool(ctx, &toolRequest.CallToolParams)
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

		for _, toolConfig := range hmc.opts.AvailableTools {
			if tool.Name == toolConfig.ToolId.Name && tool.ServerName == toolConfig.ToolId.ServerName {
				serverTools = append(serverTools, tool)
				break
			}
		}
	}
	return serverTools, nil
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
