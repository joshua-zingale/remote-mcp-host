package remotemcphost

import (
	"context"
	"strings"
	"testing"
)

func TestNewMcpHost(t *testing.T) {
	ctx := context.Background()

	host, err := NewMcpHost(ctx, strings.NewReader("![../test_servers/greetings] go run greetings.go"))

	if err != nil {
		t.Fatalf("could not create new host: %s", err)
	}

	if len(host.sessions) != 1 {
		t.Fatalf("new host has %d sessions but should have %d", len(host.sessions), 1)
	}

}

func TestListAllTools(t *testing.T) {
	ctx := context.Background()
	host, _ := NewMcpHost(ctx, strings.NewReader("![../test_servers/greetings] go run greetings.go"))

	tools, err := host.ListAllTools(ctx)

	if err != nil {
		t.Fatalf("could not list all tools: %s", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool but %d tool(s) found", len(tools))
	}
	tool := tools[0]
	if tool.Name != "greet" {
		t.Fatalf("expected test to be named 'greet' but was named '%s'", tool.Name)
	}
}
