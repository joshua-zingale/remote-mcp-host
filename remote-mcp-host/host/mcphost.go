package host

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

// Generates a new message from the input message history.
func (h *McpHost) Generate(ctx context.Context, messages []api.Message, opts *HostGenerateOptions) (api.Message, error) {
	if opts == nil {
		opts = &HostGenerateOptions{}
	}

	var parts = make([]api.UnionPart, 0)
	for {
		res, err := h.agent.Generate(ctx, messages, &agent.GenerateOptions{GeneratedParts: parts})
		if err != nil {
			return api.Message{}, err
		}
		parts = append(parts, res.Parts...)

		for _, toolRequest := range res.ToolRequests {
			session, err := h.GetClientSession(toolRequest.Name)
			if err != nil {
				return api.Message{}, err
			}
			toolRes, err := session.CallTool(ctx, &toolRequest.CallToolParams)
			if err != nil {
				parts = append(parts, api.UnionPart{Part: api.NewToolUsePartError(toolRequest.Arguments, err.Error(), api.ToolId{Name: toolRequest.Name, ServerName: toolRequest.ServerName})})
			} else {
				parts = append(parts, api.UnionPart{Part: api.NewToolUsePart(toolRequest.Arguments, *toolRes, api.ToolId{Name: toolRequest.Name, ServerName: toolRequest.ServerName})})
			}

		}

		if res.Stop {
			break
		}
	}

	return api.Message{
		Parts: parts,
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
func (h *McpHost) GetClientSession(name string) (*mcp.ClientSession, error) {
	session, ok := h.sessions[name]
	if !ok {
		return session, fmt.Errorf("invalid ClientSession name: %s", name)
	}
	return session, nil
}

// Lists all tools for a server that has an open session with this host
func (h *McpHost) ListAllTools(ctx context.Context, serverName string) ([]mcp.Tool, error) {
	session, err := h.GetClientSession(serverName)

	if err != nil {
		return []mcp.Tool{}, err
	}
	if session.InitializeResult().Capabilities.Tools == nil {
		return []mcp.Tool{}, nil
	}

	var tools []mcp.Tool

	var cursor string = ""
	for {
		res, err := session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return []mcp.Tool{}, err
		}
		for _, tool := range res.Tools {
			tools = append(tools, *tool)
		}
		cursor = res.NextCursor
		if cursor == "" {
			break
		}
	}

	return tools, nil
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
