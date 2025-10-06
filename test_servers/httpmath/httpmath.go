package main

import (
	"context"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Input struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type Output struct {
	Sum float64 `json:"sum" jsonschema:"the sum of 'a' and 'b'"`
}

func Add(ctx context.Context, req *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	return nil, Output{Sum: input.A + input.B}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "add", Description: "Adds two numbers"}, Add)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	if err := http.ListenAndServe("127.0.0.1:8080", handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
