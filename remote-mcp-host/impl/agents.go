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

var GEMINI_MAX_REQUESTS_PER_ACT int = 3

func (a GeminiAgent) Act(ctx context.Context, client agent.McpClient, messages []api.Message, opts *agent.GenerateOptions) (*agent.GenerateResult, error) {
	var generatedParts []api.UnionPart
	res, err := a.generate(ctx, client, messages, []api.UnionPart{}, &geminiConfig{})
	if err != nil {
		return nil, err
	}
	generatedParts = append(generatedParts, res.Parts...)

	for i := 1; i < GEMINI_MAX_REQUESTS_PER_ACT && res.NumToolsCalled > 0; i++ {

		if i == GEMINI_MAX_REQUESTS_PER_ACT-1 {
			res, err = a.generate(ctx, nullClient{}, messages, generatedParts, &geminiConfig{
				SystemInstruction: "The Responses from the tool calls are not visible to the user. Continue your response to the user based on the tool results in natural language. You must conclude your message to the user with this message.",
			})
		} else {
			res, err = a.generate(ctx, client, messages, generatedParts, &geminiConfig{
				SystemInstruction: "The Responses from the tool calls are not visible to the user. Continue your response to the user based on the tool results in natural language. You may call additional tools, but only if necessary.",
			})
		}

		if err != nil {
			return nil, err
		}

		generatedParts = append(generatedParts, res.Parts...)
	}

	return &agent.GenerateResult{
		Message: api.NewModelMessage(generatedParts),
	}, nil
}

func (a GeminiAgent) generate(ctx context.Context, client agent.McpClient, messages []api.Message, ammendedParts []api.UnionPart, config *geminiConfig) (*geminiGenerateResult, error) {
	if config == nil {
		config = &geminiConfig{}
	}

	combinedMessages := messages
	if len(ammendedParts) > 0 {
		combinedMessages = append(messages, *api.NewModelMessage(ammendedParts))

	}
	contents, err := messagesToGeminiContents(combinedMessages)
	if err != nil {
		return nil, err
	}

	serverTools, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	tools := serverToolsToGeminiTools(serverTools)

	res, err := a.client.Models.GenerateContent(ctx, a.opts.model, contents, &genai.GenerateContentConfig{
		Tools: tools,
	})
	if err != nil {
		return nil, fmt.Errorf("getting response from Gemini: %s", err)
	}

	var parts []api.UnionPart

	var fullText string
	{
		var bldr strings.Builder
		for _, part := range res.Candidates[0].Content.Parts {
			bldr.WriteString(part.Text)
		}
		fullText = bldr.String()
	}

	if len(fullText) > 0 {
		parts = append(parts, api.ToUnion(api.NewTextPart(fullText)))
	}

	numToolsCalled := 0
	for _, call := range res.FunctionCalls() {
		numToolsCalled += 1
		toolRequest, err := geminiFunctionCallToServerToolRequest(call)
		if err != nil {
			return nil, err
		}

		res, err := client.CallTool(ctx, toolRequest)
		if err != nil {
			parts = append(parts, api.ToUnion(api.NewToolUsePartError(toolRequest.Arguments, err.Error(), res.ToolId)))
			continue
		}
		parts = append(parts, api.ToUnion(api.NewToolUsePart(toolRequest.Arguments, res.Output, res.ToolId)))
	}

	return &geminiGenerateResult{
		Parts:          parts,
		NumToolsCalled: numToolsCalled,
	}, nil
}

type geminiGenerateResult struct {
	Parts          []api.UnionPart
	NumToolsCalled int
}

type geminiConfig struct {
	SystemInstruction string
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

type nullClient struct{}

func (n nullClient) ListTools(ctx context.Context) ([]*agent.ServerTool, error) {
	return []*agent.ServerTool{}, nil
}

func (n nullClient) CallTool(ctx context.Context, toolRequest *agent.ServerToolRequest) (*api.ToolUsePart, error) {
	return nil, fmt.Errorf("no tools are available")
}

func messagesToGeminiContents(messages []api.Message) ([]*genai.Content, error) {

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
				parts = append(parts, genai.NewPartFromFunctionCall(name, args))
				contents = append(contents, &genai.Content{
					Parts: parts,
					Role:  message.Role,
				})

				parts = make([]*genai.Part, 0)

				contents = append(contents, &genai.Content{
					Parts: []*genai.Part{genai.NewPartFromFunctionResponse(name, output)},
					Role:  "user",
				})

			default:
				return nil, fmt.Errorf("invalid part type '%v'", part)
			}
		}

		if len(parts) > 0 {
			contents = append(contents, &genai.Content{
				Parts: parts,
				Role:  message.Role,
			})
		}

	}

	return contents, nil
}

func serverToolToGeminiTool(serverTool *agent.ServerTool) *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Description: serverTool.Description,
			Name: composeToolName(
				api.ToolId{Name: serverTool.Name, ServerName: serverTool.ServerName}),
			ParametersJsonSchema: serverTool.InputSchema,
			ResponseJsonSchema:   serverTool.OutputSchema}}}
}

func serverToolsToGeminiTools(serverTools []*agent.ServerTool) []*genai.Tool {
	var tools []*genai.Tool

	for _, t := range serverTools {
		tools = append(tools, serverToolToGeminiTool(t))
	}

	return tools
}

func geminiFunctionCallToServerToolRequest(call *genai.FunctionCall) (*agent.ServerToolRequest, error) {

	toolId, err := toolIdFromCompositeName(call.Name)
	if err != nil {
		return nil, fmt.Errorf("syntactically invalid toolId used '%s'", call.Name)
	}
	return &agent.ServerToolRequest{
		ServerName: toolId.ServerName,
		CallToolParams: mcp.CallToolParams{
			Name:      toolId.Name,
			Arguments: call.Args,
		},
	}, nil
}

func composeToolName(toolId api.ToolId) string {
	return toolId.ServerName + "." + toolId.Name
}

func toolIdFromCompositeName(name string) (*api.ToolId, error) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return nil, fmt.Errorf("invalid composite tool name '%s'", name)
	}
	return &api.ToolId{
		ServerName: parts[0],
		Name:       parts[1],
	}, nil
}
