package remotemcphost

import (
	"encoding/json"
	"net/http"
)

func NewRemoteMcpMux(host *mcpHost) *http.ServeMux {

	if host == nil {
		panic("The MCP Host cannot be a null pointer")
	}

	addMcpHost := func(f func(*mcpHost, http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			f(host, w, r)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /servers/", addMcpHost(getHandleServers))

	return mux
}

func getHandleServers(host *mcpHost, w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	listing := McpServerListing{
		Servers: host.ListServerNames(),
	}

	listingJson, _ := json.Marshal(listing)

	w.Write(listingJson)
}
