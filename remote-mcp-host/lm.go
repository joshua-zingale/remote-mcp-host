package remotemcphost

import "github.com/modelcontextprotocol/go-sdk/mcp"

type LanguageModel interface {
	// Completes text.
	// The messages should be ordered from oldest to newest.
	// The options may be null, in which case default values should be used.
	Generate([]Message, *GenerateOptions) (*GenerateResult, error)
}

type GenerateOptions struct {
	Tools []*ServerTool

	// The parts already generated for the response
	// If this is non empty, then the language model should continue
	// to build on the parts
	GeneratedParts []UnionPart
}

type ServerTool struct {
	ServerName string
	mcp.Tool
}

type GenerateResult struct {
	Parts        []UnionPart
	ToolRequests []ToolRequest
	Stop         bool
}

type ToolRequest struct {
	ServerName string
	mcp.CallToolParams
}

type GeminiLM struct {
}

func (lm GeminiLM) Generate(messages []Message, opts *GenerateOptions) (*GenerateResult, error) {
	text := "nothing to echo"
	if len(messages) > 0 && len(messages[len(messages)-1].Parts) > 0 {
		if tp, ok := messages[len(messages)-1].Parts[len(messages[len(messages)-1].Parts)-1].Part.(TextPart); ok {
			text = tp.Text
		}
	}
	return &GenerateResult{
		Parts: []UnionPart{{Part: NewTextPart(text)}},
		Stop:  true,
	}, nil
}
