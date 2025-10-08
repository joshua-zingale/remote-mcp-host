package server

import (
	"net/http"

	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/agent"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/api"
	"github.com/joshua-zingale/remote-mcp-host/remote-mcp-host/host"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewRemoteMcpMux(host *host.McpHost, agent agent.Agent) *http.ServeMux {

	if host == nil {
		panic("The MCP Host cannot be a null pointer")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /servers", toJson(getServers, host, false))
	mux.HandleFunc("GET /servers/{name}/tools", toJson(getServerTools, host, false))
	mux.HandleFunc("POST /generations", toJson(postGenerations, hostAndAgent{
		host:  host,
		agent: agent,
	}, true))

	return mux
}

func postGenerations(req api.GenerationRequest, hostAndAgent hostAndAgent, r *http.Request) (api.GenerationResponse, error) {

	var toolConfigs []*api.ToolConfig

	for _, conf := range req.ToolConfigs {
		toolConfigs = append(toolConfigs, &conf)
	}

	client, err := hostAndAgent.host.GetClient(r.Context(), &host.ClientOptions{
		ToolConfigs: toolConfigs,
	})
	if err != nil {
		return api.GenerationResponse{}, err
	}

	res, err := hostAndAgent.agent.Act(r.Context(), client, req.Messages, nil)
	if err != nil {
		return api.GenerationResponse{}, err
	}
	return api.GenerationResponse{Message: *res.Message}, err
}

func getServers(_ noBody, host *host.McpHost, _ *http.Request) (api.McpServerList, error) {
	var list []api.McpServerListing
	for _, name := range host.ListServerNames() {
		list = append(list, api.McpServerListing{Name: name})
	}
	return api.McpServerList{
		Servers: list,
	}, nil
}

func getServerTools(_ interface{}, host *host.McpHost, r *http.Request) (api.ToolList, error) {
	name := r.PathValue("name")
	session, err := host.GetSession(r.Context(), name)
	if err != nil {
		return api.ToolList{}, err
	}
	if session.InitializeResult().Capabilities.Tools == nil {
		return api.ToolList{}, nil
	}

	var tools []mcp.Tool

	var cursor string = ""
	for {
		res, err := session.ListTools(r.Context(), &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return api.ToolList{}, err
		}
		for _, tool := range res.Tools {
			tools = append(tools, *tool)
		}
		cursor = res.NextCursor
		if cursor == "" {
			break
		}
	}

	return api.ToolList{
		Tools: tools,
	}, nil
}
