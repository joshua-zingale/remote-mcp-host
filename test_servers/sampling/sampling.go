package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Input struct {
	Message string `json:"name" jsonschema:"The message to start the ping pang ponging"`
}

type Output struct {
	Message string `json:"greeting" jsonschema:"The ping-pang-ponged message"`
}

func InputPingPangPong(ctx context.Context, req *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	res, err := req.Session.CreateMessage(ctx, &mcp.CreateMessageParams{
		MaxTokens: 4,
		Messages:  []*mcp.SamplingMessage{{Role: "user", Content: &mcp.TextContent{Text: "Respond only with the word \"Pang\"."}}},
	})
	if err != nil {
		return nil, Output{}, err
	}
	content := res.Content.(*mcp.TextContent)
	return nil, Output{Message: input.Message + "Ping" + content.Text + "Pong"}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "sampling", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "input ping", Description: "pings some pongs"}, InputPingPangPong)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
