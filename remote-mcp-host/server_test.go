package remotemcphost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerListing(t *testing.T) {
	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

	host, _ := NewMcpHost(ctx, client, strings.NewReader("![../test_servers/greetings][greetings] go run greetings.go"))

	mux := NewRemoteMcpMux(&host)
	r := httptest.NewRequest("GET", "/servers/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
	var listing McpServerListing
	json.NewDecoder(res.Body).Decode(&listing)

	if len(listing.Servers) != 1 {
		t.Errorf("Expected 1 MCP server(s) but found %d", len(listing.Servers))
	} else if listing.Servers[0] != "greetings" {
		t.Errorf("Expected the MCP server(s) to have names %v but had names %v", []string{"greetings"}, listing.Servers)
	}

}
