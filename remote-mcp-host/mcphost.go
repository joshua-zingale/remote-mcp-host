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

type mcpHost struct {
	sessions []*mcp.ClientSession
}

func NewMcpHost(ctx context.Context, config io.Reader) (mcpHost, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	if client == nil {
		return mcpHost{}, fmt.Errorf("could not initialize new mcp client")
	}

	sessions, err := loadSessionsFromConfig(client, ctx, config)
	if err != nil {
		return mcpHost{}, err
	}

	return mcpHost{
		sessions: sessions,
	}, nil
}

func (h *mcpHost) ListAllTools(ctx context.Context) ([]*mcp.Tool, error) {
	var tools []*mcp.Tool

	for _, s := range h.sessions {
		res, err := s.ListTools(ctx, nil)
		if err != nil {
			return tools, err
		}

		tools = append(tools, res.Tools...)
		for res.NextCursor != "" {
			listToolsParams := mcp.ListToolsParams{
				Cursor: res.NextCursor,
			}

			res, err = s.ListTools(ctx, &listToolsParams)
			if err != nil {
				return tools, err
			}
			tools = append(tools, res.Tools...)
		}

	}
	return tools, nil
}

func loadSessionsFromConfig(client *mcp.Client, ctx context.Context, r io.Reader) ([]*mcp.ClientSession, error) {
	scanner := bufio.NewScanner(r)

	var sessions []*mcp.ClientSession

	for scanner.Scan() {
		session, err := sessionFromLine(client, ctx, scanner.Text())
		if err != nil {
			return sessions, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

var stdioRegex = regexp.MustCompile(`^!\[([^\]]+)\]\s*(\S+)\s*(.*)$`)

func sessionFromLine(client *mcp.Client, ctx context.Context, line string) (*mcp.ClientSession, error) {
	err := fmt.Errorf("invalid line in config: %s", line)
	if matches := stdioRegex.FindStringSubmatch(line); len(matches) > 0 {
		command := matches[2]
		arguments := splitIntoWords(matches[3])
		cmd := exec.Command(command, arguments...)
		cmd.Dir = matches[1]

		transport := &mcp.CommandTransport{Command: cmd}
		return client.Connect(ctx, transport, nil)

	}

	return nil, err
}

var whiteSpaceRegex = regexp.MustCompile(`\s+`)

func splitIntoWords(line string) []string {
	words := whiteSpaceRegex.Split(strings.TrimSpace(line), -1)
	return words
}
