# Remote MCP

Remote MCP is a remote [MCP Host](https://modelcontextprotocol.io/specification/2025-06-18/architecture),
a web server that serves as an endpoint to generate language-model responses powered by MCP features.

## Schema
```typescript
type Role = "user" | "model";

type Part = TextPart | ToolUsePart 

interface TextPart {
    error?: string
    text?: string
    type: "text"
}

interface ToolUsePart {
    error?: string
    input: Record<string, any>
    output?: any
    toolName: string
    type: "tool-use"

}

interface Message {
    content?: Part[]
    error?: string
    role: Role
}

interface GenerationRequest {
    availableTools?: string[] // If specified, limits the tools that can be used
    messages: Message[]
}

interface GenerationResponse {
    message: Message
}
```

