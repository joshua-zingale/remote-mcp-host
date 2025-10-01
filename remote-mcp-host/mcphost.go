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
	client   *mcp.Client
	sessions map[string]*mcp.ClientSession
}

func NewMcpHost(ctx context.Context, client *mcp.Client, config io.Reader) (mcpHost, error) {

	sessions, err := loadSessionsFromConfig(client, ctx, config)
	if err != nil {
		return mcpHost{}, err
	}

	return mcpHost{
		client:   client,
		sessions: sessions,
	}, nil
}

func (h *mcpHost) ListServerNames() []string {
	keys := make([]string, 0, len(h.sessions))
	for k := range h.sessions {
		keys = append(keys, k)
	}
	return keys
}

// Gets a session with a particular name
func (h *mcpHost) GetClientSession(name string) (*mcp.ClientSession, error) {
	session, ok := h.sessions[name]
	if !ok {
		return session, fmt.Errorf("invalid ClientSession name: %s", name)
	}
	return session, nil
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
