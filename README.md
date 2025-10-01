# Remote MCP

Remote MCP is a remote [MCP Host](https://modelcontextprotocol.io/specification/2025-06-18/architecture),
a web server that serves as an endpoint to generate language-model responses powered by MCP features.

## Schema



### GET /servers
Responds with `McpServerList`

```typescript

interface McpServerListing {
    name: string
}

interface McpServerList {
    servers: McpServerListing
}

```

### POST /generations
Receives a `GenerationRequest` and responds with a `GenerationResponse`.
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
    parts?: Part[]
    error?: string
    role: Role
}

interface ToolId {
    name: string
    serverName: string
}

interface GenerationRequest {
    availableTools?: ToolId[] // If specified, limits the tools that can be used
    messages: Message[]
}

interface GenerationResponse {
    message: Message
}
```

