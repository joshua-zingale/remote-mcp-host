package remotemcphost

import (
	"context"
	"sort"
	"strings"
	"testing"
)

func TestNewMcpHost(t *testing.T) {
	ctx := context.Background()
	host, err := NewMcpHost(nil)

	if err != nil {
		t.Fatalf("could not create new host: %s", err)
	}

	host.AddSessionsFromConfig(ctx, strings.NewReader("![../test_servers/greetings] go run greetings.go"), nil)

	if len(host.sessions) != 1 {
		t.Fatalf("new host has %d sessions but should have %d", len(host.sessions), 1)
	}

}

func TestMultipleServers(t *testing.T) {
	ctx := context.Background()

	host, _ := NewMcpHost(nil)
	host.AddSessionsFromConfig(ctx, strings.NewReader("![../test_servers/greetings][greeter-1] go run greetings.go\n![../test_servers/greetings][greeter-2] go run greetings.go"), nil)

	names := host.ListServerNames()

	if len(names) != 2 {
		t.Fatalf("expected 2 tool(s) but %d tool(s) found", len(names))
	}

	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})

	if names[0] != "greeter-1" {
		t.Fatalf("greeter-1 not added to server properly")
	}
	if names[1] != "greeter-2" {
		t.Fatalf("greeter-2 not added to server properly")
	}

	session, _ := host.GetClientSession("greeter-1")
	tools, _ := session.ListTools(ctx, nil)
	if tools.Tools[0].Name != "greet" {
		t.Fatalf("greeter-1's tool not added")
	}

	session2, _ := host.GetClientSession("greeter-2")
	tools2, _ := session2.ListTools(ctx, nil)
	if tools2.Tools[0].Name != "greet" {
		t.Fatalf("greeter-2's tool not added")
	}

}
