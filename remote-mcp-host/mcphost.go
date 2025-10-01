package remotemcphost

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type clientSessionWithName struct {
	session     *mcp.ClientSession
	sessionName string
}

type mcpHost struct {
	sessions      map[string]*mcp.ClientSession
	defaultClient *mcp.Client
	opts          *McpHostOptions
}

type McpHostOptions struct {
	Lm LanguageModel
}

func NewMcpHost(opts *McpHostOptions) (mcpHost, error) {
	if opts == nil {
		opts = &McpHostOptions{
			Lm: GeminiLM{},
		}
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "Remote MCP Host Client", Version: "0.1.0"}, nil)

	return mcpHost{
		sessions:      make(map[string]*mcp.ClientSession),
		defaultClient: client,
		opts:          opts,
	}, nil
}

// Generates a new message from the input message history.
func (h *mcpHost) Generate(ctx context.Context, messages []Message, opts *HostGenerateOptions) (Message, error) {
	if opts == nil {
		opts = &HostGenerateOptions{}
	}

	var parts = make([]UnionPart, 0)
	for {
		res, err := h.opts.Lm.Generate(messages, &GenerateOptions{GeneratedParts: parts})
		if err != nil {
			return Message{}, err
		}
		parts = append(parts, res.Parts...)

		for _, toolRequest := range res.ToolRequests {
			session, err := h.GetClientSession(toolRequest.Name)
			if err != nil {
				return Message{}, err
			}
			toolRes, err := session.CallTool(ctx, &toolRequest.CallToolParams)
			if err != nil {
				parts = append(parts, UnionPart{NewToolUsePartError(toolRequest.Arguments, err.Error(), ToolId{Name: toolRequest.Name, ServerName: toolRequest.ServerName})})
			} else {
				parts = append(parts, UnionPart{NewToolUsePart(toolRequest.Arguments, *toolRes, ToolId{Name: toolRequest.Name, ServerName: toolRequest.ServerName})})
			}

		}

		if res.Stop {
			break
		}
	}

	return Message{
		Parts: parts,
		Role:  "model",
	}, nil
}

type HostGenerateOptions struct{}

// Opens MCP sessions with servers for this host.
// If a client is not specified, the host's default client is used.
func (h *mcpHost) AddSessionsFromConfig(ctx context.Context, config io.Reader, client *mcp.Client) error {
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

func (h *mcpHost) ListServerNames() []string {
	keys := make([]string, 0, len(h.sessions))
	for k := range h.sessions {
		keys = append(keys, k)
	}
	return keys
}

// Gets a session for an MCP server with a particular name
func (h *mcpHost) GetClientSession(name string) (*mcp.ClientSession, error) {
	session, ok := h.sessions[name]
	if !ok {
		return session, fmt.Errorf("invalid ClientSession name: %s", name)
	}
	return session, nil
}

// Lists all tools for a server that has an open session with this host
func (h *mcpHost) ListAllTools(ctx context.Context, serverName string) ([]mcp.Tool, error) {
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

func sessionFromLine(client *mcp.Client, ctx context.Context, line string) (clientSessionWithName, error) {
	err := fmt.Errorf("invalid line in config: %s", line)
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

	}

	return clientSessionWithName{}, err
}

var whiteSpaceRegex = regexp.MustCompile(`\s+`)

func splitIntoWords(line string) []string {
	words := whiteSpaceRegex.Split(strings.TrimSpace(line), -1)
	return words
}
