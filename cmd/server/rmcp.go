package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/host"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/impl"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/server"
)

func main() {

	ctx := context.Background()

	McpHost, err := host.NewMcpHost(nil)
	if err != nil {
		panic(err)
	}

	err = McpHost.AddSessionsFromConfig(ctx, strings.NewReader("![./test_servers/greetings][greetings] go run greetings.go"), nil)
	if err != nil {
		panic(err)
	}

	agent, err := impl.NewGeminiAgent(ctx, nil)
	if err != nil {
		panic(err)
	}
	mux := server.NewRemoteMcpMux(&McpHost, agent)

	server := http.Server{
		Handler: mux,
	}

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
