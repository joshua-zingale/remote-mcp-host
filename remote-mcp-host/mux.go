package remotemcphost

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewRemoteMcpMux(host *mcpHost) *http.ServeMux {

	if host == nil {
		panic("The MCP Host cannot be a null pointer")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /servers", toJson(getServers, host, false))
	mux.HandleFunc("GET /servers/{name}/tools", toJson(getServerTools, host, false))
	mux.HandleFunc("POST /generations", toJson(postGenerations, host, true))

	return mux
}

func postGenerations(req GenerationRequest, host *mcpHost, r *http.Request) (GenerationResponse, error) {

	message, err := host.Generate(r.Context(), req.Messages, nil)

	return GenerationResponse{Message: message}, err
}

func getServers(_ NoBody, host *mcpHost, _ *http.Request) (McpServerList, error) {
	var list []McpServerListing
	for _, name := range host.ListServerNames() {
		list = append(list, McpServerListing{Name: name})
	}
	return McpServerList{
		Servers: list,
	}, nil
}

func getServerTools(_ interface{}, host *mcpHost, r *http.Request) (ToolList, error) {
	name := r.PathValue("name")
	session, err := host.GetClientSession(name)
	if err != nil {
		return ToolList{}, err
	}
	if session.InitializeResult().Capabilities.Tools == nil {
		return ToolList{}, nil
	}

	var tools []mcp.Tool

	var cursor string = ""
	for {
		res, err := session.ListTools(r.Context(), &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return ToolList{}, err
		}
		for _, tool := range res.Tools {
			tools = append(tools, *tool)
		}
		cursor = res.NextCursor
		if cursor == "" {
			break
		}
	}

	return ToolList{
		Tools: tools,
	}, nil
}
