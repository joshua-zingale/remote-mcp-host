package impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/agent"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/genai"
)

type GeminiAgent struct {
	client *genai.Client
	opts   *GeminiOpts
}

func (a GeminiAgent) Generate(ctx context.Context, messages []api.Message, opts *agent.GenerateOptions) (*agent.GenerateResult, error) {

	var contents []*genai.Content

	for _, message := range messages {
		var parts []*genai.Part

		for _, part := range message.Parts {

			switch part := part.Part.(type) {
			case api.TextPart:
				parts = append(parts, genai.NewPartFromText(part.Text))
			case api.ToolUsePart:
				name := composeToolName(part.ToolId)
				args, ok := part.Input.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid type for input arguments of tool use '%v'", part.Input)
				}
				output, ok := part.Output.StructuredContent.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid type for output arguments of tool use '%v'", part.Output.StructuredContent)
				}
				parts = append(parts, genai.NewPartFromFunctionCall(name, args), genai.NewPartFromFunctionResponse(name, output))
			default:
				return nil, fmt.Errorf("invalid part type '%v'", part)
			}
		}
		contents = append(contents, &genai.Content{
			Parts: parts,
			Role:  message.Role,
		})
	}

	res, err := a.client.Models.GenerateContent(ctx, a.opts.model, contents, &genai.GenerateContentConfig{})
	if err != nil {
		return nil, err
	}

	var toolRequests []agent.ToolRequest
	for _, call := range res.FunctionCalls() {
		toolId, err := toolIdFromCompositeName(call.Name)
		if err != nil {
			return nil, fmt.Errorf("syntactically invalid toolId used '%s'", call.Name)
		}
		toolRequests = append(toolRequests, agent.ToolRequest{
			ServerName: toolId.ServerName,
			CallToolParams: mcp.CallToolParams{
				Name:      toolId.Name,
				Arguments: call.Args,
			},
		})
	}

	return &agent.GenerateResult{
		Parts:        []api.UnionPart{{Part: api.NewTextPart(res.Text())}},
		ToolRequests: toolRequests,
		Continue:     false,
	}, nil
}

func NewGeminiAgent(ctx context.Context, opts *GeminiOpts) (*GeminiAgent, error) {

	if opts == nil {
		opts = &GeminiOpts{model: "gemini-2.0-flash"}
	}

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &GeminiAgent{
		client: client,
		opts:   opts,
	}, nil
}

type GeminiOpts struct {
	model string
}

func composeToolName(toolId api.ToolId) string {
	return toolId.ServerName + "/" + toolId.Name
}

func toolIdFromCompositeName(name string) (*api.ToolId, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return nil, fmt.Errorf("Invalid composite tool name '%s', name")
	}
	return &api.ToolId{
		ServerName: parts[0],
		Name:       parts[1],
	}, nil
}
