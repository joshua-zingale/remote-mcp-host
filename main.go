package main

import (
	"context"
	"net/http"
	"strings"

	host "github.com/joshua-zingale/remote-mcp-host/remote-mcp-host"
)

func main() {

	ctx := context.Background()
	mcpHost, err := host.NewMcpHost(nil)
	if err != nil {
		panic(err)
	}

	mcpHost.AddSessionsFromConfig(ctx, strings.NewReader("![./test_servers/greetings][greetings] go run greetings.go\n![./test_servers/sampling][sampling] go run sampling.go"), nil)

	mux := host.NewRemoteMcpMux(&mcpHost)

	server := http.Server{
		Handler: mux,
	}

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
