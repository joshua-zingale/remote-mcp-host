package api

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type McpServerList struct {
	Servers []McpServerListing `json:"servers"`
}

type McpServerListing struct {
	Name string `json:"name"`
}

type ToolList struct {
	Tools []mcp.Tool `json:"tools"`
}

type RoleType = string

type TextPart struct {
	Error string `json:"error,omitempty"`
	Text  string `json:"text,omitempty"`
	Type  string `json:"type"`
}

type ToolUsePart struct {
	Error  string             `json:"error,omitempty"`
	Input  any                `json:"input"`
	Output mcp.CallToolResult `json:"output,omitempty"`
	ToolId ToolId             `json:"toolId"`
	Type   string             `json:"type"`
}

type Message struct {
	Parts []UnionPart `json:"parts"`
	Role  RoleType    `json:"role"`
}

type ToolId struct {
	Name       string `json:"name"`
	ServerName string `json:"serverName"`
}

type ToolConfig struct {
	ToolId ToolId `json:"toolId"`
	// Arguments any    `json:"arguments"`
}

type GenerationRequest struct {
	AvailableTools []ToolConfig `json:"availableTools,omitempty"`
	Messages       []Message    `json:"messages"`
}

type GenerationResponse struct {
	Message Message `json:"message"`
}

type Part interface {
	PartType() string
}

// UnionPart is the wrapper type used for marshaling and unmarshaling the union.
type UnionPart struct {
	Part
}

func (up UnionPart) MarshalJSON() ([]byte, error) {

	return json.Marshal(up.Part)
}

func (up *UnionPart) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	switch temp.Type {
	case "text":
		var p TextPart
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		up.Part = p
	case "tool-use":
		var p ToolUsePart
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		up.Part = p
	default:
		return fmt.Errorf("unknown part type: %s", temp.Type)
	}
	return nil
}

func NewRole(role string) (RoleType, error) {
	switch role {
	case "user", "model":
		return role, nil
	default:
		return role, fmt.Errorf("invalid role: %s", role)
	}
}

func NewTextPart(text string) TextPart {
	return TextPart{
		Text: text,
		Type: "text",
	}
}

func NewTextPartError(errorText string) TextPart {
	return TextPart{
		Error: errorText,
		Type:  "text",
	}
}

func (t TextPart) PartType() string {
	return "text"
}

func NewToolUsePart(input any, output mcp.CallToolResult, toolId ToolId) ToolUsePart {
	return ToolUsePart{
		Input:  input,
		Output: output,
		ToolId: toolId,
		Type:   "tool-use",
	}
}

func NewToolUsePartError(input any, errorText string, toolId ToolId) ToolUsePart {
	return ToolUsePart{
		Input:  input,
		Error:  errorText,
		ToolId: toolId,
		Type:   "tool-use",
	}
}

func NewModelMessage(parts []UnionPart) *Message {
	return &Message{
		Parts: parts,
		Role:  "model",
	}
}

func (t ToolUsePart) PartType() string {
	return "tool-use"
}
