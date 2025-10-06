package testutil

import (
	agent "github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/Agent"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/api"
)

type EchoAgent struct {
}

func (lm EchoAgent) Generate(messages []api.Message, opts *agent.GenerateOptions) (*agent.GenerateResult, error) {
	text := "nothing to echo"
	if len(messages) > 0 && len(messages[len(messages)-1].Parts) > 0 {
		if tp, ok := messages[len(messages)-1].Parts[len(messages[len(messages)-1].Parts)-1].Part.(api.TextPart); ok {
			text = tp.Text
		}
	}
	return &agent.GenerateResult{
		Parts: []api.UnionPart{{Part: api.NewTextPart(text)}},
		Stop:  true,
	}, nil
}
