package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/joshua-zingale/remote-mcp-host/internal/testutil"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/host"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/server"
)

func main() {

	ctx := context.Background()
	McpHost, err := host.NewMcpHost(testutil.EchoAgent{}, nil)
	if err != nil {
		panic(err)
	}

	McpHost.AddSessionsFromConfig(ctx, strings.NewReader("![./test_servers/greetings][greetings] go run greetings.go\n![./test_servers/sampling][sampling] go run sampling.go"), nil)

	mux := server.NewRemoteMcpMux(&McpHost)

	server := http.Server{
		Handler: mux,
	}

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
