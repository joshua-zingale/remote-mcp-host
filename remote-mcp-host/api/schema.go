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

type ToolPatch struct {
	// If non-nil, overwrites conflicting portions of the Input for this
	// with values specified here.
	// For example, setting Input = map[string]int{"a": 3} will force
	// the argument for "a" always to be 3.
	// The actual input is overwritten where it diverges from the
	// Input specified here. In the previous example, other
	// input values beside "a" would be unchanged.
	//
	// Setting this will also remove any specified arguments from the Schema.
	Input map[string]any `json:"Input,omitempty"`
}

type ToolConfig struct {
	ToolId    ToolId    `json:"toolId"`
	ToolPatch ToolPatch `json:"toolPatch,omitempty"`
}

type GenerationRequest struct {
	ToolConfigs            []ToolConfig `json:"toolConfigs,omitempty"`
	Messages               []Message    `json:"messages"`
	OnlyUseConfiguredTools bool         `json:"onlyIncludeConfiguredTools,omitempty"`
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

func ToUnion(part Part) UnionPart {
	return UnionPart{Part: part}
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
